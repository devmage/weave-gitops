import _ from "lodash";
import * as React from "react";
import { useRouteMatch } from "react-router-dom";
import styled from "styled-components";
import { AppContext } from "../contexts/AppContext";
import { useListAutomations, useSyncFluxObject } from "../hooks/automations";
import { useToggleSuspend } from "../hooks/flux";
import { FluxObjectKind } from "../lib/api/core/types.pb";
import { Source } from "../lib/objects";
import Alert from "./Alert";
import AutomationsTable from "./AutomationsTable";
import Button from "./Button";
import EventsTable from "./EventsTable";
import Flex from "./Flex";
import InfoList, { InfoField } from "./InfoList";
import LoadingPage from "./LoadingPage";
import Metadata from "./Metadata";
import PageStatus from "./PageStatus";
import Spacer from "./Spacer";
import SubRouterTabs, { RouterTab } from "./SubRouterTabs";
import SyncButton from "./SyncButton";
import Text from "./Text";
import YamlView from "./YamlView";

type Props = {
  className?: string;
  type: FluxObjectKind;
  children?: JSX.Element;
  source: Source;
  info: InfoField[];
};

function SourceDetail({ className, source, info, type }: Props) {
  const { notifySuccess } = React.useContext(AppContext);
  const { data: automations, isLoading: automationsLoading } =
    useListAutomations();
  const { path } = useRouteMatch();
  const [isSuspended, setIsSuspended] = React.useState(false);

  const suspend = useToggleSuspend(
    {
      name: source.name,
      namespace: source.namespace,
      clusterName: source.clusterName,
      kind: type,
      suspend: !isSuspended,
    },
    "sources"
  );

  const sync = useSyncFluxObject({
    name: source.name,
    namespace: source.namespace,
    clusterName: source.clusterName,
    kind: type,
  });

  if (automationsLoading) {
    return <LoadingPage />;
  }

  if (isSuspended != source.suspended) {
    setIsSuspended(source.suspended);
  }

  const isNameRelevant = (expectedName) => {
    return expectedName == source.name;
  };

  const isRelevant = (expectedType, expectedName) => {
    return expectedType == source.kind && isNameRelevant(expectedName);
  };

  const relevantAutomations = _.filter(automations?.result, (a) => {
    if (!source) {
      return false;
    }
    if (a.clusterName != source.clusterName) {
      return false;
    }

    if (
      type == FluxObjectKind.KindHelmChart &&
      isNameRelevant(a?.helmChart?.name)
    ) {
      return true;
    }

    return (
      isRelevant(a?.sourceRef?.kind, a?.sourceRef?.name) ||
      isRelevant(a?.helmChart?.sourceRef?.kind, a?.helmChart?.sourceRef?.name)
    );
  });

  const handleSyncClicked = () => {
    sync.mutateAsync({ withSource: false }).then(() => {
      notifySuccess("Resource synced successfully");
    });
  };

  return (
    <Flex wide tall column className={className}>
      <Text size="large" semiBold titleHeight>
        {source.name}
      </Text>
      {suspend.error && (
        <Alert severity="error" title="Error" message={suspend.error.message} />
      )}
      <PageStatus conditions={source.conditions} suspended={source.suspended} />
      <Flex wide start>
        <SyncButton
          onClick={handleSyncClicked}
          loading={sync.isLoading}
          disabled={source.suspended}
          hideDropdown={true}
        />
        <Spacer padding="xs" />
        <Button
          onClick={() => suspend.mutateAsync()}
          loading={suspend.isLoading}
        >
          {isSuspended ? "Resume" : "Suspend"}
        </Button>
      </Flex>

      <SubRouterTabs rootPath={`${path}/details`}>
        <RouterTab name="Details" path={`${path}/details`}>
          <>
            <InfoList items={info} />
            <Metadata metadata={source.metadata} />
            <AutomationsTable automations={relevantAutomations} hideSource />
          </>
        </RouterTab>
        <RouterTab name="Events" path={`${path}/events`}>
          <EventsTable
            namespace={source.namespace}
            involvedObject={{
              kind: source.kind,
              name: source.name,
              namespace: source.namespace,
            }}
          />
        </RouterTab>
        <RouterTab name="yaml" path={`${path}/yaml`}>
          <YamlView
            yaml={source.yaml}
            object={{
              kind: source.kind,
              name: source.name,
              namespace: source.namespace,
            }}
          />
        </RouterTab>
      </SubRouterTabs>
    </Flex>
  );
}

export default styled(SourceDetail).attrs({ className: SourceDetail.name })`
  ${PageStatus} {
    padding: ${(props) => props.theme.spacing.small} 0px;
  }
  ${SubRouterTabs} {
    margin-top: ${(props) => props.theme.spacing.medium};
  }
`;
