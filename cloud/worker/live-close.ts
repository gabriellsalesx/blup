// Closing a WebSocket with a code that only a receiver may see throws.
/** The code to answer a close with: one a WebSocket may send. */
export const closeCodeToSend=(code:number)=>code===1000||(code>=3000&&code<=4999)||(code>=1001&&code<=1014&&code!==1004&&code!==1005&&code!==1006)?code:1000;
