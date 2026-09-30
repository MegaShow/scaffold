import { AxiosClient, type AxiosInstance } from './axios.ts';
import {
  type Client,
  HTTPError,
  type HeaderValue,
  type Method,
  type QueryValue,
  type Request,
  type Response,
} from './client.ts';
import { AdaptComposer, FormComposer, JsonComposer, type RequestComposer } from './composer.ts';
import { FetchClient, type FetchClientOptions } from './fetch.ts';
import { JsonResolver, type ResponseResolver } from './resolver.ts';

export type {
  AxiosInstance,
  Client,
  FetchClientOptions,
  HeaderValue,
  Method,
  QueryValue,
  Request,
  RequestComposer,
  Response,
  ResponseResolver,
};

export { AdaptComposer, AxiosClient, FetchClient, FormComposer, HTTPError, JsonComposer, JsonResolver };
