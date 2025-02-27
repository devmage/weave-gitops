import { MuiThemeProvider } from "@material-ui/core";
import { createMemoryHistory } from "history";
import _ from "lodash";
import * as React from "react";
import { QueryClient, QueryClientProvider } from "react-query";
import { Router } from "react-router-dom";
import { ThemeProvider } from "styled-components";
import AppContextProvider, { AppProps } from "../contexts/AppContext";
import CoreClientContextProvider from "../contexts/CoreClientContext";
import {
  Applications,
  GetGithubAuthStatusRequest,
  GetGithubAuthStatusResponse,
  GetGithubDeviceCodeRequest,
  GetGithubDeviceCodeResponse,
  ParseRepoURLRequest,
  ParseRepoURLResponse,
  ValidateProviderTokenRequest,
  ValidateProviderTokenResponse,
} from "./api/applications/applications.pb";
import {
  Core,
  GetChildObjectsRequest,
  GetChildObjectsResponse,
  GetReconciledObjectsRequest,
  GetReconciledObjectsResponse,
  GetVersionRequest,
  GetVersionResponse,
} from "./api/core/core.pb";
import theme, { muiTheme } from "./theme";
import { RequestError } from "./types";

export type CoreOverrides = {
  GetChildObjects?: (req: GetChildObjectsRequest) => GetChildObjectsResponse;
  GetReconciledObjects?: (
    req: GetReconciledObjectsRequest
  ) => GetReconciledObjectsResponse;
  GetVersion?: (req: GetVersionRequest) => GetVersionResponse;
};

export const createCoreMockClient = (
  ovr: CoreOverrides,
  error?: RequestError
): typeof Core => {
  const promisified = _.reduce(
    ovr,
    (result, handlerFn, method) => {
      result[method] = (req) => {
        if (error) {
          return new Promise((_, reject) => reject(error));
        }
        return new Promise((accept) => accept(handlerFn(req) as any));
      };

      return result;
    },
    {}
  );

  return promisified as typeof Core;
};

export type ApplicationOverrides = {
  GetGithubDeviceCode?: (
    req: GetGithubDeviceCodeRequest
  ) => GetGithubDeviceCodeResponse;
  GetGithubAuthStatus?: (
    req: GetGithubAuthStatusRequest
  ) => GetGithubAuthStatusResponse;
  ParseRepoURL?: (req: ParseRepoURLRequest) => ParseRepoURLResponse;
  ValidateProviderToken?: (
    req: ValidateProviderTokenRequest
  ) => ValidateProviderTokenResponse;
};

// Don't make the user wire up all the promise stuff to be interface-compliant
export const createMockClient = (
  ovr: ApplicationOverrides,
  error?: RequestError
): typeof Applications => {
  const promisified = _.reduce(
    ovr,
    (result, handlerFn, method) => {
      result[method] = (req) => {
        if (error) {
          return new Promise((_, reject) => reject(error));
        }
        return new Promise((accept) => accept(handlerFn(req) as any));
      };

      return result;
    },
    {}
  );

  return promisified as typeof Applications;
};

export function withTheme(element) {
  return (
    <MuiThemeProvider theme={muiTheme}>
      <ThemeProvider theme={theme}>{element}</ThemeProvider>
    </MuiThemeProvider>
  );
}

type TestContextProps = AppProps & { api?: Core };

export function withContext(
  TestComponent,
  url: string,
  { api = {}, ...appProps }: TestContextProps
) {
  const history = createMemoryHistory();
  history.push(url);
  const queryClient = new QueryClient();
  const isElement = React.isValidElement(TestComponent);
  return (
    <Router history={history}>
      <AppContextProvider renderFooter {...appProps}>
        <QueryClientProvider client={queryClient}>
          <CoreClientContextProvider api={api as any}>
            {isElement ? TestComponent : <TestComponent />}
          </CoreClientContextProvider>
        </QueryClientProvider>
      </AppContextProvider>
    </Router>
  );
}
