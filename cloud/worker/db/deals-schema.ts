import {sqliteTable,text,integer,index,primaryKey,uniqueIndex} from "drizzle-orm/sqlite-core";

// Blup Deals. Sources and offers are shared by every account (each source
// is read once per round for everyone); channels, store IDs, hunts and the
// send queue belong to one account. Times are milliseconds, money is in the
// currency's minor unit.

// Where the posts go: a Telegram channel or a Discord channel the Blup bot
// was added to. target is the chat or channel id.
export const dealsChannel=sqliteTable("deals_channel",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),platform:text("platform").notNull(),name:text("name").notNull(),target:text("target").notNull(),guildId:text("guild_id"),
 active:integer("active").notNull().default(1),verifiedAt:integer("verified_at"),createdAt:integer("created_at").notNull(),
 planDisabled:integer("plan_disabled").notNull().default(0),retireAt:integer("retire_at"),
 // target is sealed (like a Pulse destination); targetHash finds the same
 // chat again (the bot removed, the same webhook twice). nextAt keeps the
 // interval between two posts in the channel; lastError is the last
 // delivery code.
 targetHash:text("target_hash"),nextAt:integer("next_at").notNull().default(0),lastError:text("last_error"),
 // The language the channel's posts go out in (null: the account's); a
 // post in another language is translated by the AI.
 language:text("language"),
 // A WhatsApp Canal: the account's WhatsApp connection (deals_whatsapp)
 // that posts in it.
 sessionId:text("session_id"),
 // When the panel may send examples to it again (one round a minute; on
 // WhatsApp, the Canal's turn).
 testAt:integer("test_at").notNull().default(0),
},t=>[index("deals_channel_owner_idx").on(t.userId,t.id),uniqueIndex("deals_channel_target_idx").on(t.userId,t.platform,t.target),index("deals_channel_retire_idx").on(t.retireAt),index("deals_channel_hash_idx").on(t.targetHash)]);

// The account's ID or tag at each store (matt_word, tag...) and, for the
// stores searched with the account's own API key, that key sealed like a
// Pulse destination (never shown back).
export const dealsAffiliate=sqliteTable("deals_affiliate",{
 userId:text("user_id").notNull(),store:text("store").notNull(),tag:text("tag"),credentials:text("credentials"),level:text("level"),createdAt:integer("created_at").notNull(),updatedAt:integer("updated_at").notNull(),
 // What the panel may show: a few characters of the ID, never the secret.
 hint:text("hint"),
},t=>[primaryKey({columns:[t.userId,t.store]})]);

// A shared source: a feed (GamerPower), a store catalog (Nuuvem) or a
// mirrored public channel. key identifies it (telegram:<channel>), so two
// accounts mirroring the same channel share one read. cursor is where the
// last read stopped.
export const dealsSource=sqliteTable("deals_source",{
 id:text("id").primaryKey(),kind:text("kind").notNull(),key:text("key").notNull().unique(),config:text("config").notNull().default("{}"),
 cursor:text("cursor"),nextAt:integer("next_at").notNull(),lastAt:integer("last_at"),failures:integer("failures").notNull().default(0),error:text("error"),
 claim:text("claim"),claimUntil:integer("claim_until").notNull().default(0),createdAt:integer("created_at").notNull(),
},t=>[index("deals_source_due_idx").on(t.nextAt)]);

// An account's hunt: what it wants from a source (filters as JSON) and the
// channels that receive it (JSON list of deals_channel ids). kind mirror
// counts toward the mirror limit.
export const dealsHunt=sqliteTable("deals_hunt",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),name:text("name").notNull(),kind:text("kind").notNull(),sourceId:text("source_id"),
 filters:text("filters").notNull().default("{}"),channels:text("channels").notNull().default("[]"),intervalMinutes:integer("interval_minutes").notNull().default(30),
 active:integer("active").notNull().default(1),lastPostAt:integer("last_post_at"),createdAt:integer("created_at").notNull(),
 planDisabled:integer("plan_disabled").notNull().default(0),retireAt:integer("retire_at"),
},t=>[index("deals_hunt_owner_idx").on(t.userId,t.id),index("deals_hunt_source_idx").on(t.sourceId),index("deals_hunt_retire_idx").on(t.retireAt)]);

// The global catalog: one row per offer, whoever found it. hash (store and
// product, normalized) keeps the same offer from coming in twice.
export const dealsOffer=sqliteTable("deals_offer",{
 id:text("id").primaryKey(),hash:text("hash").notNull().unique(),store:text("store").notNull(),externalId:text("external_id"),sourceId:text("source_id"),
 title:text("title").notNull(),url:text("url").notNull(),imageUrl:text("image_url"),price:integer("price"),originalPrice:integer("original_price"),currency:text("currency"),
 firstSeenAt:integer("first_seen_at").notNull(),lastSeenAt:integer("last_seen_at").notNull(),expiresAt:integer("expires_at"),
 // kind: product (a pasted link), video (a TikTok video's news),
 // game_free (GamerPower), nuuvem_deal (Nuuvem's catalog) or store_deal
 // (a store hunt's find, Shopee or AliExpress). platforms:
 // JSON list of the panel's platform ids.
 // discount: percent off. credit: the source the post must credit.
 kind:text("kind").notNull().default("product"),platforms:text("platforms").notNull().default("[]"),discount:integer("discount"),credit:text("credit"),
 // What the AI made of the product, once for everyone: a short name and
 // its category (shared/deals-post.ts dealCategories).
 shortTitle:text("short_title"),category:text("category"),
},t=>[index("deals_offer_store_idx").on(t.store,t.externalId),index("deals_offer_seen_idx").on(t.lastSeenAt),index("deals_offer_source_idx").on(t.sourceId,t.lastSeenAt),index("deals_offer_kind_idx").on(t.kind,t.firstSeenAt)]);

// Price history, for an honest discount (the price before the sale, not
// the one the store shows crossed out).
export const dealsPrice=sqliteTable("deals_price",{
 offerId:text("offer_id").notNull(),at:integer("at").notNull(),price:integer("price").notNull(),currency:text("currency").notNull(),
},t=>[primaryKey({columns:[t.offerId,t.at]}),index("deals_price_at_idx").on(t.at)]);

// The send queue (an outbox, like Pulse's alerts). One offer goes to a
// channel once; day (YYYY-MM-DD, UTC) counts the posts per day. code is the
// short link that counts clicks.
export const dealsPost=sqliteTable("deals_post",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),huntId:text("hunt_id"),channelId:text("channel_id").notNull(),offerId:text("offer_id").notNull(),
 code:text("code").notNull().unique(),link:text("link").notNull(),state:text("state").notNull().default("queued"),attempts:integer("attempts").notNull().default(0),
 nextAt:integer("next_at").notNull(),claim:text("claim"),claimUntil:integer("claim_until").notNull().default(0),day:text("day").notNull(),
 sentAt:integer("sent_at"),error:text("error"),createdAt:integer("created_at").notNull(),
 // Whether link carries the person's affiliate ID (the post says so).
 affiliate:integer("affiliate").notNull().default(0),
 // 0 when the offer's price is another country's (only the discount goes).
 priceShown:integer("price_shown").notNull().default(1),
 // 1 when it waits its turn in the account's pace (deals_setting.pace).
 paced:integer("paced").notNull().default(0),
 // What the person wrote (or asked the AI for) about the product, and the
 // coupon, both for this post only.
 description:text("description"),coupon:text("coupon"),
},t=>[uniqueIndex("deals_post_once_idx").on(t.channelId,t.offerId),index("deals_post_due_idx").on(t.state,t.nextAt),index("deals_post_owner_day_idx").on(t.userId,t.day),index("deals_post_created_idx").on(t.createdAt),
 // A deals page lists its channel's sent posts, newest first.
 index("deals_post_channel_sent_idx").on(t.channelId,t.state,t.sentAt)]);

// Clicks on the short links, added up per post and day (few writes).
export const dealsClickDay=sqliteTable("deals_click_day",{
 postId:text("post_id").notNull(),userId:text("user_id").notNull(),day:text("day").notNull(),clicks:integer("clicks").notNull().default(0),
},t=>[primaryKey({columns:[t.postId,t.day]}),index("deals_click_owner_day_idx").on(t.userId,t.day)]);

// Linking a Telegram channel or group: the person opens the bot in private
// (/start code), then adds the bot as admin where the posts go; that chat
// becomes the candidate the panel confirms.
export const dealsTelegramLink=sqliteTable("deals_telegram_link",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),hash:text("hash").notNull().unique(),state:text("state").notNull().default("pending"),expiresAt:integer("expires_at").notNull(),
 telegramUser:text("telegram_user"),chatId:text("chat_id"),chatLabel:text("chat_label"),channelId:text("channel_id"),
},t=>[index("deals_telegram_link_owner_idx").on(t.userId),index("deals_telegram_link_user_idx").on(t.telegramUser,t.state)]);
// Telegram updates already applied (it may send one twice).
export const dealsTelegramUpdate=sqliteTable("deals_telegram_update",{
 id:integer("id").primaryKey(),receivedAt:integer("received_at").notNull(),applied:integer("applied").notNull().default(0),
},t=>[index("deals_telegram_update_received_idx").on(t.receivedAt)]);
// Whether an account has had Deals, to follow it when the plan goes (like
// pulse_account): readOnlyUntil is set the first time the plan is missing
// (14 days to read and export), then the private Deals data is purged.
export const dealsAccount=sqliteTable("deals_account",{
 userId:text("user_id").primaryKey(),lastPlan:text("last_plan").notNull(),readOnlyUntil:integer("read_only_until"),purgeStarted:integer("purge_started"),
},t=>[index("deals_account_read_only_idx").on(t.readOnlyUntil)]);
// How an account's posts go out (one setting for all its channels): pace,
// minutes between two queued posts (0: as soon as found), and paceLast,
// the time the last one was given; droppedAt, when a hunt last found more
// than the pace can send in two days (the panel warns). headlines (JSON,
// per kind, missing: the default line), the extra text and the social
// networks (JSON [{network,url}]), each with its switch; style, how the
// post is laid out (classic, or emoji: a line and a mark for each part);
// aiTitles and aiDescribe, whether the AI shortens the names and writes a
// description in the hunts' posts (from the plan's AI allowance).
export const dealsSetting=sqliteTable("deals_setting",{
 userId:text("user_id").primaryKey(),pace:integer("pace").notNull().default(0),paceLast:integer("pace_last").notNull().default(0),droppedAt:integer("dropped_at"),
 headlines:text("headlines").notNull().default("{}"),extra:text("extra"),extraOn:integer("extra_on").notNull().default(0),
 socials:text("socials").notNull().default("[]"),socialsOn:integer("socials_on").notNull().default(0),style:text("style").notNull().default("classic"),aiTitles:integer("ai_titles").notNull().default(0),aiDescribe:integer("ai_describe").notNull().default(0),updatedAt:integer("updated_at").notNull().default(0),
});

// The account's WhatsApp, connected to the Bridge (cloud/bridge) by QR: id
// is the session's name at the Bridge, which keeps the keys (only them).
// state is the Bridge's (pairing, connected, logged_out, banned...),
// phoneEnd the number's last four digits; the person accepted the risk of
// the unofficial way at riskAcceptedAt. One per account; each of its
// Canais is a deals_channel with this sessionId.
export const dealsWhatsapp=sqliteTable("deals_whatsapp",{
 id:text("id").primaryKey(),userId:text("user_id").notNull(),state:text("state").notNull().default("pairing"),phoneEnd:text("phone_end"),
 riskAcceptedAt:integer("risk_accepted_at").notNull(),createdAt:integer("created_at").notNull(),updatedAt:integer("updated_at").notNull(),
},t=>[uniqueIndex("deals_whatsapp_user_idx").on(t.userId)]);

// The account's public deals page (a channel of type "site"): the deals
// sent to channelId show at deals.getblup.site/<slug>/, or on the account's
// own domain. One per account. logoLight and logoDark are the versions of
// the logos kept in R2 (DEALS_ASSETS), null without one.
export const dealsSite=sqliteTable("deals_site",{
 id:text("id").primaryKey(),userId:text("user_id").notNull().unique(),channelId:text("channel_id").notNull().unique(),slug:text("slug").notNull().unique(),
 title:text("title").notNull(),description:text("description").notNull().default(""),language:text("language").notNull().default("pt"),
 logoLight:text("logo_light"),logoDark:text("logo_dark"),createdAt:integer("created_at").notNull(),updatedAt:integer("updated_at").notNull(),
 // The header, as a Pulse page's: the name beside the logo or not, and up
 // to five links (JSON, as navLinks).
 showTitle:integer("show_title").notNull().default(1),links:text("links").notNull().default("[]"),
 // Its social networks (JSON, one of each of siteNetworks).
 socials:text("socials").notNull().default("[]"),
});

// A deals page's own domain, verified like a Pulse page's (TXT proof, then
// a Cloudflare custom hostname).
export const dealsSiteDomain=sqliteTable("deals_site_domain",{
 siteId:text("site_id").primaryKey(),hostname:text("hostname").notNull(),proof:text("proof").notNull(),providerId:text("provider_id"),verifiedAt:integer("verified_at"),
 status:text("status").notNull().default("pending_ownership"),ssl:text("ssl").notNull().default("pending"),records:text("records").notNull().default("[]"),checkedAt:integer("checked_at"),
},t=>[index("deals_site_domain_host_idx").on(t.hostname,t.verifiedAt)]);
