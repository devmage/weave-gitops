package clustersmngr_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/weaveworks/weave-gitops/core/clustersmngr"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/server/auth"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/fields"
)

func TestClientGet(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)

	clusterName := "mycluster"

	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	kust := &kustomizev1.Kustomization{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}
	ctx := context.Background()
	g.Expect(k8sEnv.Client.Create(ctx, kust)).To(Succeed())

	k := &kustomizev1.Kustomization{}

	g.Expect(clustersClient.Get(ctx, clusterName, types.NamespacedName{Name: appName, Namespace: ns.Name}, k)).To(Succeed())
	g.Expect(k.Name).To(Equal(appName))
}

func TestClientClusteredList(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)
	namespaced := true

	clusterName := "mycluster"
	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	kust := &kustomizev1.Kustomization{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}
	ctx := context.Background()
	g.Expect(k8sEnv.Client.Create(ctx, kust)).To(Succeed())

	cklist := clustersmngr.NewClusteredList(func() client.ObjectList {
		return &kustomizev1.KustomizationList{}
	})

	g.Expect(clustersClient.ClusteredList(ctx, cklist, namespaced)).To(Succeed())

	klist := cklist.Lists()[clusterName][0].(*kustomizev1.KustomizationList)

	g.Expect(klist.Items).To(HaveLen(1))
	g.Expect(klist.Items[0].Name).To(Equal(appName))

	gitRepo := &sourcev1.GitRepository{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: sourcev1.GitRepositorySpec{
			URL: "https://example.com/repo",
			SecretRef: &meta.LocalObjectReference{
				Name: "somesecret",
			},
		},
	}

	g.Expect(k8sEnv.Client.Create(ctx, gitRepo)).To(Succeed())

	cgrlist := clustersmngr.NewClusteredList(func() client.ObjectList {
		return &sourcev1.GitRepositoryList{}
	})

	g.Expect(clustersClient.ClusteredList(ctx, cgrlist, namespaced)).To(Succeed())

	glist := cgrlist.Lists()[clusterName][0].(*sourcev1.GitRepositoryList)
	g.Expect(glist.Items).To(HaveLen(1))
	g.Expect(glist.Items[0].Name).To(Equal(appName))
}

func TestClientClusteredListPagination(t *testing.T) {
	g := NewGomegaWithT(t)
	ctx := context.Background()
	ns1 := createNamespace(g)
	ns2 := createNamespace(g)
	namespaced := true

	clusterName := "mycluster"

	createKust := func(name string, nsName string) {
		kust := &kustomizev1.Kustomization{
			ObjectMeta: v1.ObjectMeta{
				Name:      name,
				Namespace: nsName,
			},
			Spec: kustomizev1.KustomizationSpec{
				SourceRef: kustomizev1.CrossNamespaceSourceReference{
					Kind: "GitRepository",
				},
			},
		}
		ctx := context.Background()
		g.Expect(k8sEnv.Client.Create(ctx, kust)).To(Succeed())
	}

	// Create 2 kustomizations in 2 namespaces
	for i := 0; i < 2; i++ {
		appName := "myapp-" + strconv.Itoa(i)
		createKust(appName, ns1.Name)
	}

	for i := 0; i < 1; i++ {
		appName := "myapp-" + strconv.Itoa(i)
		createKust(appName, ns2.Name)
	}

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns1, *ns2},
	}
	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	// First request comes with no continue token
	cklist := clustersmngr.NewClusteredList(func() client.ObjectList {
		return &kustomizev1.KustomizationList{}
	})
	g.Expect(clustersClient.ClusteredList(ctx, cklist, namespaced, client.Limit(1), client.Continue(""))).To(Succeed())
	g.Expect(cklist.Lists()[clusterName]).To(HaveLen(2))
	klist := cklist.Lists()[clusterName][0].(*kustomizev1.KustomizationList)
	g.Expect(klist.Items).To(HaveLen(1))

	continueToken := cklist.GetContinue()

	// Second request comes with the continue token
	cklist = clustersmngr.NewClusteredList(func() client.ObjectList {
		return &kustomizev1.KustomizationList{}
	})
	g.Expect(clustersClient.ClusteredList(ctx, cklist, namespaced, client.Limit(1), client.Continue(continueToken))).To(Succeed())
	g.Expect(cklist.Lists()[clusterName]).To(HaveLen(1))
	klist0 := cklist.Lists()[clusterName][0].(*kustomizev1.KustomizationList)
	g.Expect(klist0.Items).To(HaveLen(1))

	continueToken = cklist.GetContinue()

	// Third request comes with an empty namespaces continue token
	cklist = clustersmngr.NewClusteredList(func() client.ObjectList {
		return &kustomizev1.KustomizationList{}
	})
	g.Expect(clustersClient.ClusteredList(ctx, cklist, namespaced, client.Limit(1), client.Continue(continueToken))).To(Succeed())
	g.Expect(cklist.Lists()[clusterName]).To(HaveLen(0))
}

func TestClientClusteredListClusterScoped(t *testing.T) {
	g := NewGomegaWithT(t)

	clusterName := "mycluster"
	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)
	clusterRole := rbacv1.ClusterRole{
		ObjectMeta: v1.ObjectMeta{
			Name: appName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Verbs:     []string{"get"},
				Resources: []string{"pods"},
			},
		},
	}
	opts := []client.ListOption{&client.ListOptions{FieldSelector: fields.OneTermEqualSelector("metadata.name", appName)}}

	ctx := context.Background()
	g.Expect(k8sEnv.Client.Create(ctx, &clusterRole)).To(Succeed())

	cklist := clustersmngr.NewClusteredList(func() client.ObjectList {
		return &rbacv1.ClusterRoleList{}
	})

	g.Expect(clustersClient.ClusteredList(ctx, cklist, false, opts...)).To(Succeed())

	klist := cklist.Lists()[clusterName][0].(*rbacv1.ClusterRoleList)

	g.Expect(klist.Items).To(HaveLen(1))
	g.Expect(klist.Items[0].Name).To(Equal(appName))
}

func TestClientCLusteredListErrors(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)

	clusterName := "mycluster"

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	cklist := clustersmngr.NewClusteredList(func() client.ObjectList {
		return &kustomizev1.KustomizationList{}
	})

	labels := client.MatchingLabels{
		"foo": "@invalid",
	}

	cerr := clustersClient.ClusteredList(context.Background(), cklist, true, labels)
	g.Expect(cerr).ToNot(BeNil())

	var errs clustersmngr.ClusteredListError

	g.Expect(errors.As(cerr, &errs)).To(BeTrue())
}

func TestClientList(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)

	clusterName := "mycluster"
	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	kust := &kustomizev1.Kustomization{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}
	ctx := context.Background()
	g.Expect(k8sEnv.Client.Create(ctx, kust)).To(Succeed())

	list := &kustomizev1.KustomizationList{}

	g.Expect(clustersClient.List(ctx, clusterName, list, client.InNamespace(ns.Name))).To(Succeed())

	g.Expect(list.Items).To(HaveLen(1))
	g.Expect(list.Items[0].Name).To(Equal(appName))
}

func TestClientCreate(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)

	clusterName := "mycluster"
	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	kust := &kustomizev1.Kustomization{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}
	ctx := context.Background()

	g.Expect(clustersClient.Create(ctx, clusterName, kust)).To(Succeed())

	k := &kustomizev1.Kustomization{}
	g.Expect(clustersClient.Get(ctx, clusterName, types.NamespacedName{Name: appName, Namespace: ns.Name}, k)).To(Succeed())
	g.Expect(k.Name).To(Equal(appName))
}

func TestClientDelete(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)

	clusterName := "mycluster"
	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	kust := &kustomizev1.Kustomization{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: kustomizev1.KustomizationSpec{
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}
	ctx := context.Background()

	g.Expect(k8sEnv.Client.Create(ctx, kust)).To(Succeed())

	g.Expect(clustersClient.Delete(ctx, clusterName, kust)).To(Succeed())
}

func TestClientUpdate(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)

	clusterName := "mycluster"
	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	kust := &kustomizev1.Kustomization{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: kustomizev1.KustomizationSpec{
			Path: "/foo",
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}
	ctx := context.Background()
	g.Expect(k8sEnv.Client.Create(ctx, kust)).To(Succeed())

	kust.Spec.Path = "/bar"
	g.Expect(clustersClient.Update(ctx, clusterName, kust)).To(Succeed())

	k := &kustomizev1.Kustomization{}
	g.Expect(k8sEnv.Client.Get(ctx, types.NamespacedName{Name: appName, Namespace: ns.Name}, k)).To(Succeed())
	g.Expect(k.Spec.Path).To(Equal("/bar"))
}

func TestClientPatch(t *testing.T) {
	g := NewGomegaWithT(t)
	ns := createNamespace(g)

	clusterName := "mycluster"
	appName := "myapp" + rand.String(5)

	clientsPool := createClusterClientsPool(g, clusterName)

	nsMap := map[string][]corev1.Namespace{
		clusterName: {*ns},
	}

	clustersClient := clustersmngr.NewClient(clientsPool, nsMap)

	kust := &kustomizev1.Kustomization{
		TypeMeta: metav1.TypeMeta{
			Kind:       kustomizev1.KustomizationKind,
			APIVersion: kustomizev1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: ns.Name,
		},
		Spec: kustomizev1.KustomizationSpec{
			Path: "/foo",
			SourceRef: kustomizev1.CrossNamespaceSourceReference{
				Kind: "GitRepository",
			},
		},
	}

	ctx := context.Background()
	opt := []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner("test"),
	}
	g.Expect(clustersClient.Patch(ctx, clusterName, kust, client.Apply, opt...)).To(Succeed())

	k := &kustomizev1.Kustomization{}
	g.Expect(k8sEnv.Client.Get(ctx, types.NamespacedName{Name: appName, Namespace: ns.Name}, k)).To(Succeed())
	g.Expect(k.Spec.Path).To(Equal("/foo"))
}

func createNamespace(g *GomegaWithT) *corev1.Namespace {
	ns := &corev1.Namespace{}
	ns.Name = "kube-test-" + rand.String(5)

	g.Expect(k8sEnv.Client.Create(context.Background(), ns)).To(Succeed())

	return ns
}

func createClusterClientsPool(g *GomegaWithT, clusterName string) clustersmngr.ClientsPool {
	scheme, err := kube.CreateScheme()
	g.Expect(err).To(BeNil())

	clientsPool := clustersmngr.NewClustersClientsPool(scheme)

	err = clientsPool.Add(
		// Put the user in the `system:masters` group to avoid auth errors
		clustersmngr.ClientConfigWithUser(&auth.UserPrincipal{ID: "anne", Groups: []string{"system:masters"}}),
		clustersmngr.Cluster{
			Name:      clusterName,
			Server:    k8sEnv.Rest.Host,
			TLSConfig: k8sEnv.Rest.TLSClientConfig,
		},
	)

	g.Expect(err).To(BeNil())

	return clientsPool
}
