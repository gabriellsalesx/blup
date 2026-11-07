import { monthOf } from "./pulse/limits";
import { DAY, dayOf } from "./deals/store";
import type { Plan } from "./plans";

// What an account uses of its add-ons, read only (no entitlement is
// written): Blup Pulse's projects, monitors, pages and this month's errors,
// and Blup Deals' channels, hunts, mirrors and today's posts. The products'
// own panels and the staff read these same numbers.

export type PulseCounts = { projects: number; monitors: number; pages: number; month: string; accepted: number; reserved: number; bytes: number };
export type DealsCounts = { channels: number; hunts: number; mirrors: number; postsToday: number; resetsAt: string };

const live = "(retire_at IS NULL OR retire_at>?2)";

// Errors accepted this month come from each project's Durable Object; the
// reserved count (D1) is what the quota already holds.
export async function pulseCountsOf(env: Cloudflare.Env, userId: string, readable = true): Promise<PulseCounts> {
	const month = monthOf(), now = Date.now();
	const [projects, row, reserved] = await Promise.all([
		env.DB.prepare("SELECT object_name AS objectName FROM pulse_project WHERE user_id=? AND deleted_at IS NULL ORDER BY id LIMIT 10").bind(userId).all<{ objectName: string }>(),
		env.DB.prepare(`SELECT
			(SELECT count(*) FROM pulse_project WHERE user_id=?1 AND deleted_at IS NULL AND ${live}) AS projects,
			(SELECT count(*) FROM pulse_monitor WHERE user_id=?1 AND ${live}) AS monitors,
			(SELECT count(*) FROM pulse_page WHERE user_id=?1 AND ${live}) AS pages`).bind(userId, now).first<{ projects: number; monitors: number; pages: number }>(),
		env.DB.prepare("SELECT reserved FROM pulse_month WHERE user_id=? AND month=?").bind(userId, month).first<{ reserved: number }>(),
	]);
	const stats = readable ? await Promise.all(projects.results.map((p) => env.PULSE_PROJECT.getByName(p.objectName).usage(month))) : [];
	return {
		projects: row?.projects ?? 0, monitors: row?.monitors ?? 0, pages: row?.pages ?? 0, month,
		accepted: stats.reduce((n, s) => n + s.accepted, 0), reserved: reserved?.reserved ?? 0, bytes: stats.reduce((n, s) => n + s.bytes, 0),
	};
}

export async function dealsCountsOf(db: D1Database, userId: string, now = Date.now()): Promise<DealsCounts> {
	const row = await db.prepare(`SELECT
		(SELECT count(*) FROM deals_channel WHERE user_id=?1 AND ${live}) AS channels,
		(SELECT count(*) FROM deals_hunt WHERE user_id=?1 AND kind<>'mirror' AND ${live}) AS hunts,
		(SELECT count(*) FROM deals_hunt WHERE user_id=?1 AND kind='mirror' AND ${live}) AS mirrors,
		(SELECT count(*) FROM deals_post WHERE user_id=?1 AND day=?3) AS postsToday`).bind(userId, now, dayOf(now)).first<{ channels: number; hunts: number; mirrors: number; postsToday: number }>();
	return { channels: row?.channels ?? 0, hunts: row?.hunts ?? 0, mirrors: row?.mirrors ?? 0, postsToday: row?.postsToday ?? 0, resetsAt: new Date(Math.floor(now / DAY) * DAY + DAY).toISOString() };
}

// The staff's view: each add-on the plan includes, its limits and use;
// null when the plan doesn't include it.
export async function addonUsageOf(env: Cloudflare.Env, userId: string, plan: Plan) {
	const [pulse, deals] = await Promise.all([
		plan.pulse ? pulseCountsOf(env, userId) : null,
		plan.deals ? dealsCountsOf(env.DB, userId) : null,
	]);
	return {
		pulse: plan.pulse && pulse ? { limits: plan.pulse, used: pulse } : null,
		deals: plan.deals && deals ? { limits: plan.deals, used: deals } : null,
	};
}
