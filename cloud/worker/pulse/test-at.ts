// For the tests: an item that must be there (the test fails, clearly, when
// it isn't), instead of an index that may be undefined.
export function at<T>(list:ArrayLike<T>,index=0):T{const value=list[index];if(value===undefined)throw new Error(`missing item ${index}`);return value;}
