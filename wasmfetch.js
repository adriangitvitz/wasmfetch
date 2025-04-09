class WasmFetch {
  constructor() {
    this.initiliazed = false;
    this.initPromise = this.init();
  }

  async init() {
    if (this.initiliazed) return this;
    await this.loadScript('/wasm_exec.js');
    const go = new window.Go();
    const response = await fetch('/wasmfetch.wasm');
    const bytes = await response.arrayBuffer();
    console.log(bytes);
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);

    go.run(instance);

    await this.waitForFunctions();

    this.initiliazed = true;

  }

  loadScript(src) {
    return new Promise((resolve, reject) => {
      if (document.querySelector(`script[src="${src}"]`)) {
        resolve();
        return;
      }

      const script = document.createElement('script');
      script.src = src;
      script.onload = resolve;
      script.onerror = reject;
      document.head.appendChild(script);
    });
  }

  waitForFunctions() {
    const checkFunctions = () => {
      return typeof window.goProcessJSON === 'function' &&
        typeof window.goExtractFields === 'function' &&
        typeof window.goMakeRequest === 'function';
    };

    if (checkFunctions()) return Promise.resolve();
    return new Promise(resolve => {
      const checkInterval = setInterval(() => {
        if (checkFunctions()) {
          clearInterval(checkInterval);
          resolve();
        }
      }, 100);
    });
  }

  async processJSON(jsonStr) {
    await this.initPromise;
    return window.goProcessJSON(jsonStr);
  }

  async extractFields(jsonStr, fields) {
    await this.initPromise;
    return window.goExtractFields(jsonStr, fields);
  }

  async get(url, config = {}) {
    await this.initPromise;
    return window.goMakeRequest(url, { ...config, method: 'GET' });
  }
}

const wasmfetch = new WasmFetch();
export default wasmfetch;
