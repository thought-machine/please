(async () => {
  const fuseListResp = await fetch("/fusejs_list.json");
  const fuseList = await fuseListResp.json();
  const fuse = new Fuse(fuseList, {
    includeMatches: true,
    minMatchCharLength: 2,
    ignoreLocation: true,
    useExtendedSearch: true,

    keys: [
      { name: "heading", weight: 1 },
      { name: "textContent", weight: 0.3 },
    ],
  });

  window.fuseSearch = (pattern) => fuse.search(`'${pattern}`);
})();
