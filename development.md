# Overview

- are we using caddy with https? (why not replace with httpd module?)

org case
- `<organization>.pages-test.psi.ch` -> having a __gitea-pages__ repo in organization (+topic gitea-pages ?)

Simple case
- `pages-test.psi.ch/<organization>/<repository>`

mixed case??
- `<organization>.pages-test.psi.ch/<repository>` ???? 

- Markdown support? (no?) 
- Support/read gitea-pages.toml file? (no)
- Require/read gitea-pages topic on repo ? (yes)
- need gitea-pages branch? (would use default branch)


https://medium.com/@danieljimgarcia/publishing-static-sites-to-github-pages-using-github-actions-8040f57dfeaf

https://gist.github.com/ramnathv/2227408

https://olgarithms.github.io/sphinx-tutorial/docs/7-hosting-on-github-pages.html


https://blog.logrocket.com/automatically-build-deploy-vuejs-app-github-pages/

```javascript
await execa("git", ["checkout", "--orphan", "gh-pages"]);
// eslint-disable-next-line no-console
console.log("Building started...");
await execa("npm", ["run", "build"]);
// Understand if it's dist or build folder
const folderName = fs.existsSync("dist") ? "dist" : "build";
await execa("git", ["--work-tree", folderName, "add", "--all"]);
await execa("git", ["--work-tree", folderName, "commit", "-m", "gh-pages"]);
console.log("Pushing to gh-pages...");
await execa("git", ["push", "origin", "HEAD:gh-pages", "--force"]);
await execa("rm", ["-r", folderName]);
await execa("git", ["checkout", "-f", "master"]);
await execa("git", ["branch", "-D", "gh-pages"]);
console.log("Successfully deployed, check your settings");
```
https://git-scm.com/docs/git-worktree


https://dev.to/the_one/deploy-to-github-pages-like-a-pro-with-github-actions-4hdg