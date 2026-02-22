# Video Demo

## Scene 1: Explain context

The purpose of our product is to protect against supply chain attacks in the open source ecosystem where malicious packages make it into the official registry and installed either by means of typo squatting, compromise of a legitimate package, or otherwise.

To prevent these attacks, we are building a private registry comprising of verified packages which pass through a series of automated tests to ensure they are safe.

## Scene 2: Demonstrate front-end processing

We simply provide our project's `package.json` to the platform which builds the dependency tree and starts the analysis process.

The behavior of each package is recorded and processed by an AI agent, which determines the legitimacy of the package.

Once the legitimacy of the packages is determined, the approved ones move to the secured private registry.

<!-- By this point, analysis should be done -->

## Scene 3: Talk about front-end

With the analysis now complete, let's take a look at the results. We can see that all of our dependencies have been approved.

For each one, the platform provides a behavioral summary and the AI reasoning that has lead to the decision.

From a developer or CI/CD perspective, all that is needed to use our secure registry, is a single configuration change.

<!-- Show `npm config set` -->

## Scene 4: Malicious Package

Now lets see how it performs with an actual malware sample from September, 2025.

<!-- Speed through. Show red. Click red. Show AI analysis -->

The AI agent has detected that the package attempting to exfiltrate our credentials using trufflehog, and has flagged it as malicious, thus blocking it from our registry.

<!-- npm install blocked -->

## Impact statement

What you've seen today is a fully functional and practical pipeline that proactively secures your projects, bringing enterprise-level security to any customer. Our vision extends beyond this point, bringing our technology to PyPi, Maven, and other ecosystems to secure any project.
