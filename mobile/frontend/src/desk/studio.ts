// What the Studio's screens ask of its Go side, here on the phone's services.
// The screens in this folder are the Studio's: only this file and their
// touch and layout changes differ, so they keep reading the same files.
import { BoardService, CalendarService, NotesService } from "../../bindings/github.com/gabriellsalesx/blup/mobile";
import type { DBSchema } from "../../bindings/github.com/gabriellsalesx/blup/internal/app/models";

export const ReadBoard = (rel: string) => BoardService.Read(rel);
export const SaveBoard = (rel: string, text: string) => BoardService.Save(rel, text);
export const SaveBoardImage = (rel: string, name: string, data: string) => BoardService.SaveImage(rel, name, data);
export const BoardFile = (rel: string) => BoardService.File(rel);
export const ReadNote = async (rel: string) => ({ body: await BoardService.ReadNote(rel) });
export const NoteImage = (note: string, src: string) => NotesService.Image(note, src);
export const OpenURL = (url: string) => NotesService.OpenURL(url);
export const ReadDatabase = (rel: string) => BoardService.ReadDatabase(rel);
export const SaveDatabaseSchema = (rel: string, schema: DBSchema) => BoardService.SaveDatabaseSchema(rel, schema);
export const AddRecord = (rel: string, name: string, props: Record<string, unknown>) => BoardService.AddRecord(rel, name, props);
export const SetProps = (rel: string, edits: { key: string; value: unknown }[]) => BoardService.SetProps(rel, edits);
export const DeleteItem = (rel: string) => CalendarService.Delete(rel);
export const RestoreSection = (trashed: string) => CalendarService.Restore(trashed);
