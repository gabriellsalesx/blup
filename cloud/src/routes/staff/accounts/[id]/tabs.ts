// The tabs of an account in the staff, in order (?tab=access opens one).
export const accountTabs = ["summary", "usage", "billing", "access", "history", "actions"] as const;
export type AccountTab = (typeof accountTabs)[number];
