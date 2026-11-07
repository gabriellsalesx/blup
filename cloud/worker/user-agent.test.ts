import {it,expect} from "vitest";
import {readAgent} from "../shared/user-agent";

// Where each session is: Blup Studio by its own name, a browser by its
// usual words.
it("reads Blup Studio and the browsers",()=>{
 expect(readAgent("Blup Studio (Linux; my-pc)")).toEqual({platform:"linux",system:"Linux",studio:{host:"my-pc"},browser:null});
 expect(readAgent("Blup Studio (android; localhost)")).toMatchObject({platform:"android",system:"Android"});
 expect(readAgent("Blup Studio (Plan9; box)")).toMatchObject({platform:"other",system:"Plan9"});
 expect(readAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36")).toMatchObject({platform:"linux",browser:"Chrome"});
 expect(readAgent("Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Mobile Safari/537.36")).toMatchObject({platform:"android",browser:"Chrome"});
 expect(readAgent("Mozilla/5.0 (iPhone; CPU iPhone OS 19_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/19.0 Mobile/15E148 Safari/604.1")).toMatchObject({platform:"ios",browser:"Safari"});
 expect(readAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7; rv:140.0) Gecko/20100101 Firefox/140.0")).toMatchObject({platform:"macos",browser:"Firefox"});
 expect(readAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36 Edg/155.0.0.0")).toMatchObject({platform:"windows",browser:"Edge"});
 expect(readAgent(null)).toEqual({platform:"other",system:null,studio:null,browser:null});
});
