// Where to go once the channel is linked: the first steps when they sent
// here (?next=/start...), else the channels.
export function load({url}){
 const next=url.searchParams.get("next");
 return {next:next&&/^\/start(\?[\w=&-]*)?$/.test(next)?next:null};
}
