import { describe, expect, it } from "vitest";
import { chunks, codesFromList, codesFromRefunds, csvRows } from "./appsumo-codes";

describe("AppSumo files read in the staff", () => {
	it("splits a CSV with quotes, commas and CRLF", () => {
		expect(csvRows('a,"b, c","d ""e"""\r\n1,2,3\n')).toEqual([["a", "b, c", 'd "e"'], ["1", "2", "3"]]);
	});
	it("reads one code per line, trimmed and once", () => {
		expect(codesFromList("\uFEFFBLUPAAAA1111BBBB\r\n  BLUPAAAA2222BBBB \n\nBLUPAAAA1111BBBB\n\"BLUPAAAA3333BBBB\"")).toEqual(["BLUPAAAA1111BBBB", "BLUPAAAA2222BBBB", "BLUPAAAA3333BBBB"]);
	});
	it("takes the refunds' code column, whatever its case and place", () => {
		const csv = "Order ID,Email,Redemption Code,Refunded At\n123,a@x.io,BLUPAAAA1111BBBB,2026-11-01\n124,b@x.io, BLUPAAAA2222BBBB ,2026-11-02\n125,c@x.io,,2026-11-03\n126,d@x.io,BLUPAAAA1111BBBB,2026-11-04\n";
		expect(codesFromRefunds(csv)).toEqual(["BLUPAAAA1111BBBB", "BLUPAAAA2222BBBB"]);
	});
	it("prefers the exact code column to another one with \"code\" in it", () => {
		const csv = "\uFEFFCountry Code,Redemption Code\nBR,BLUPAAAA1111BBBB\nUS,BLUPAAAA2222BBBB\n";
		expect(codesFromRefunds(csv)).toEqual(["BLUPAAAA1111BBBB", "BLUPAAAA2222BBBB"]);
		expect(codesFromRefunds("Country Code,Codes\nBR,BLUPAAAA3333BBBB\n")).toEqual(["BLUPAAAA3333BBBB"]);
	});
	it("finds hyphenated codes without a code column", () => {
		expect(codesFromRefunds("order,license\n1,BLUP-AAAA-1111-BBBB\n2,blup-aaaa-2222\n3,NOTBLUP-XXXX\n")).toEqual(["BLUP-AAAA-1111-BBBB", "blup-aaaa-2222"]);
	});
	it("without a code column, finds the tokens that look like ours", () => {
		const csv = "order,email,license\n123,ann@example.com,blupaaaa1111bbbb\n124,zz,XYZW12345678 BLUPAAAA2222BBBB\n";
		expect(codesFromRefunds(csv)).toEqual(["blupaaaa1111bbbb", "BLUPAAAA2222BBBB"]);
	});
	it("finds nothing in an empty file", () => {
		expect(codesFromRefunds("")).toEqual([]);
		expect(codesFromList("\n\n")).toEqual([]);
	});
	it("cuts a list in parts, in order", () => {
		expect(chunks([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]]);
		expect(chunks([], 100)).toEqual([]);
	});
});
