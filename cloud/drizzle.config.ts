import { defineConfig } from "drizzle-kit";

// SQL migrations for D1, applied with wrangler (npm run db:migrate:local).
export default defineConfig({
	dialect: "sqlite",
	schema: "./worker/db/schema.ts",
	out: "./migrations",
});
