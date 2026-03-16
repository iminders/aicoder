#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const zlib = require('zlib');
const tar = require('tar');

const REPO = 'iminders/aicoder';
const BINARY_NAME = 'aicoder';
const BIN_DIR = path.join(__dirname, '..', 'bin');

// Detect platform
function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  let os, archName;

  switch (platform) {
    case 'darwin':
      os = 'darwin';
      break;
    case 'linux':
      os = 'linux';
      break;
    case 'win32':
      os = 'windows';
      break;
    default:
      throw new Error(`Unsupported platform: ${platform}`);
  }

  switch (arch) {
    case 'x64':
      archName = 'x86_64';
      break;
    case 'arm64':
      archName = 'arm64';
      break;
    default:
      throw new Error(`Unsupported architecture: ${arch}`);
  }

  return { os, arch: archName, platform };
}

// Get latest release version
function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO}/releases/latest`,
      headers: {
        'User-Agent': 'aicoder-npm-installer'
      }
    };

    https.get(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        try {
          const release = JSON.parse(data);
          resolve(release.tag_name);
        } catch (err) {
          reject(err);
        }
      });
    }).on('error', reject);
  });
}

// Download file
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(url, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        // Follow redirect
        return downloadFile(res.headers.location, dest).then(resolve).catch(reject);
      }
      res.pipe(file);
      file.on('finish', () => {
        file.close();
        resolve();
      });
    }).on('error', (err) => {
      fs.unlink(dest, () => {});
      reject(err);
    });
  });
}

// Extract tar.gz
async function extractTarGz(archivePath, destDir) {
  return tar.extract({
    file: archivePath,
    cwd: destDir
  });
}

// Main installation
async function install() {
  try {
    console.log('🚀 Installing aicoder...');

    const { os, arch, platform } = getPlatform();
    console.log(`📦 Platform: ${os}_${arch}`);

    const version = await getLatestVersion();
    console.log(`📌 Version: ${version}`);

    const versionNum = version.replace('v', '');
    const ext = platform === 'win32' ? 'zip' : 'tar.gz';
    const filename = `${BINARY_NAME}_${versionNum}_${os}_${arch}.${ext}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/${version}/${filename}`;

    console.log(`⬇️  Downloading from ${downloadUrl}...`);

    const archivePath = path.join(BIN_DIR, filename);
    if (!fs.existsSync(BIN_DIR)) {
      fs.mkdirSync(BIN_DIR, { recursive: true });
    }

    await downloadFile(downloadUrl, archivePath);
    console.log('✅ Download complete');

    console.log('📂 Extracting...');
    await extractTarGz(archivePath, BIN_DIR);

    // Clean up archive
    fs.unlinkSync(archivePath);

    // Make binary executable
    const binaryPath = path.join(BIN_DIR, platform === 'win32' ? `${BINARY_NAME}.exe` : BINARY_NAME);
    if (platform !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
    }

    console.log('✅ Installation complete!');
    console.log(`\n🎉 Run 'aicoder --help' to get started`);

  } catch (err) {
    console.error('❌ Installation failed:', err.message);
    process.exit(1);
  }
}

install();
