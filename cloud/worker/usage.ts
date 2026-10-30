// What an account is using: storage of the synced notes, sites published
// through the cloud, AI credits this month, Studio devices and the last
// sync. Only numbers (the notes themselves are encrypted on the person's
// computer). Every figure is 0 or null until the sync and the cloud sites
// exist; their tables will feed this one place, which both the account
// panel and the staff read.

export type Usage = {
	storageBytes: number;
	sites: number;
	aiCreditsUsed: number;
	aiCreditsResetAt: string | null; // when this month's credits start over
	devices: number; // Blup Studio installs signed in
	lastSyncAt: string | null;
};

export async function usageOf(_userId: string, periodEnd?: Date | null): Promise<Usage> {
	return {
		storageBytes: 0,
		sites: 0,
		aiCreditsUsed: 0,
		aiCreditsResetAt: periodEnd ? periodEnd.toISOString() : null,
		devices: 0,
		lastSyncAt: null,
	};
}
