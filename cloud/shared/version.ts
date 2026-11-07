// The version of Blup Cloud, its sites and its panels: one place, the
// "version" of package.json (raise it there when publishing). It goes with
// every error sent to Blup Pulse ("blup-cloud@1.0.0"). A named import: the
// bundles take this one field, not the whole file.
import { version } from "../package.json";

export const BLUP_VERSION: string = version;
