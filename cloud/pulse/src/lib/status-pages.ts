import {type InferResponseType} from "hono/client";
import {pulseApi,parseResponse} from "./api";
// What the status page screens share: the page, the monitors that can be
// shown on it and its incidents.
export type StatusPage=InferResponseType<typeof pulseApi.pages.$get,200>["pages"][number];
export type Monitor=InferResponseType<typeof pulseApi.monitors.$get,200>["monitors"][number];
export const incidentsGET=pulseApi.pages[":page"].incidents.$get;
export type Incident=InferResponseType<typeof incidentsGET,200>["incidents"][number];
export type IncidentState=Incident["state"];
export type IncidentUpdate=InferResponseType<typeof pulseApi.pages[":page"]["incidents"][":incident"]["updates"]["$get"],200>["updates"][number];
export type ServiceStatus=Incident["services"][number]["status"]|"none";
export const getPages=async()=>(await parseResponse(pulseApi.pages.$get())).pages;
export const getMonitors=async()=>(await parseResponse(pulseApi.monitors.$get())).monitors;
export const getIncidents=async(page:string)=>(await parseResponse(incidentsGET({param:{page}}))).incidents;
