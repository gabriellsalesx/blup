// Ctrl+Z and Ctrl+Shift+Z in the Linux window: Wails (GTK) takes the keys
// before the page sees them and runs document.execCommand("undo"/"redo"),
// which the editor and the terminal don't use. Here that call goes to
// whoever has the focus (the editor's own history, the terminal's Ctrl+Z);
// fields keep the browser's undo.
type Handler = (cmd: "undo" | "redo") => boolean;
export const undoHandlers = new Set<Handler>();

export function installUndo() {
  const native = document.execCommand.bind(document);
  document.execCommand = (cmd: string, ...rest: any[]) => {
    if (cmd === "undo" || cmd === "redo") {
      for (const h of undoHandlers) if (h(cmd)) return true;
    }
    return native(cmd, ...rest);
  };
}
