// What an account uses (the API's `usage`) and how to show it.
export type Usage = {
	storageBytes: number;
	vaults: number;
	sites: number;
	devices: number;
	lastSyncAt: string | null;
};

export const bytes = (n: number) => {
	if (n === 0) return "0 MB";
	if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
	if (n < 1024 ** 3) return `${(n / 1024 ** 2).toLocaleString("pt-BR", { maximumFractionDigits: 1 })} MB`;
	return `${(n / 1024 ** 3).toLocaleString("pt-BR", { maximumFractionDigits: 1 })} GB`;
};

// A plan's storage limit is in MB.
export const megabytes = (mb: number) => (mb >= 1024 ? `${(mb / 1024).toLocaleString("pt-BR")} GB` : `${mb} MB`);
