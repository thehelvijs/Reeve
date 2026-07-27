// Vite serves a `?raw` import as the file's text. Declared locally rather than
// pulling in vite/client, which would drag Node typings in for one test.
declare module '*?raw' {
  const contents: string;
  export default contents;
}
