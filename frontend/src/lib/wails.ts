// Thin typed layer over the Wails-generated bindings (../../wailsjs), so
// components import from one place instead of reaching into the generated
// tree directly.
import {
  Cancel as CancelBinding,
  ConvertImages as ConvertImagesBinding,
  ConvertVideo as ConvertVideoBinding,
  PickImages as PickImagesBinding,
  PickVideo as PickVideoBinding,
  RevealOutput as RevealOutputBinding,
} from "../../wailsjs/go/app/App";
import type { app } from "../../wailsjs/go/models";
import {
  EventsOn,
  Quit as QuitRuntime,
  WindowMinimise as WindowMinimiseRuntime,
} from "../../wailsjs/runtime/runtime";

// DTOs mirrored from internal/app - re-exported under shorter names so
// callers never import wailsjs/go/models directly.
export type VideoInfo = app.VideoInfo;
export type ImageInfo = app.ImageInfo;
export type VideoRequest = app.VideoRequest;
export type ImagesRequest = app.ImagesRequest;
export type ConvertResult = app.ConvertResult;

// Mirrors internal/app/convert.go's ProgressEvent{Phase string; Percent int},
// delivered over the "convert:progress" Wails event. Not part of models.ts
// because it is only ever emitted, never returned from a bound call.
export interface ProgressEvent {
  Phase: string;
  Percent: number;
}

// Bound Go methods (internal/app/app.go, media.go, convert.go, preview.go).
export const PickVideo = PickVideoBinding;
export const PickImages = PickImagesBinding;
export const ConvertVideo = ConvertVideoBinding;
export const ConvertImages = ConvertImagesBinding;
export const Cancel = CancelBinding;
export const RevealOutput = RevealOutputBinding;

// Wails JS runtime - only the pieces the shell and modes need.
export const WindowMinimise = WindowMinimiseRuntime;
export const Quit = QuitRuntime;

/**
 * Subscribes to conversion progress events. Returns an unsubscribe function
 * (EventsOn already returns one; this just gives the callback a typed shape).
 */
export function onProgress(handler: (event: ProgressEvent) => void): () => void {
  return EventsOn("convert:progress", (event: ProgressEvent) => handler(event));
}
