Move `wasmfetch.wasm` to `public` and `wasmfetch.js` inside `src`

Example usage:
``` javascript
const fetchData = async () => {
    const response = await wasmfetch.get('https://jsonplaceholder.typicode.com/posts/1');
    console.log(response.data);
};
```

