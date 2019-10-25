const fs = require('fs');
const path = require('path');

function ignorePath(path) {
    let includedPaths = [
        /\/package(?:-lock)?\.json$/,
        /\/node_modules$/,
        /\/node_modules\//,
        /\/bin$/,
        /\/bin\//,
        /\/frontend$/,
        /\/frontend\/jasons-game$/,
        /\/frontend\/main\.js$/,
        /\/frontend\/update\.html$/,
        /\/frontend\/update\.css$/,
        /\/frontend\/preload\.js$/,
        /\/frontend\/restart\.html$/,
    ];

    let ignore = true;

    includedPaths.some((ip) => {
        if (path === "" || ip.test(path)) {
            ignore = false;
            return true;
        }
    });

    return ignore;
}

function deleteOtherPlatforms(forgeConfig, buildPath, electronVersion, platform, arch) {
    let ourBinary = `jasonsgame-${platform}-public`;

    if (platform === "win32") {
        ourBinary += '.exe';
    }

    let binPath = path.join(buildPath, 'bin');

    fs.readdir(binPath, (err, binaries) => {
        if (err !== null) {
            console.log(`Error getting game binaries: ${err}`);
            return;
        }

        binaries.forEach((bin) => {
            if (bin !== ourBinary) {
                fs.unlink(path.join(binPath, bin), (err) => {
                    if (err !== null) {
                        console.log(`Error deleting ${bin}: ${err}`);
                    }
                });
            }
        })
    });
}

module.exports = {
    "makers": [
        {
            "name": '@electron-forge/maker-zip',
            "platforms": ['darwin', 'linux']
        },
        {
            "name": '@electron-forge/maker-squirrel'
        }
    ],
    "packagerConfig": {
        "appBundleId": "com.quorumcontrol.jasonsgame",
        "packageManager": "npm",
        "asar": {
            "unpackDir": 'bin'
        },
        "ignore": ignorePath,
        "osxSign": {
            "hardenedRuntime": true,
            "identity": "Developer ID Application: Quorum Control GmbH (8U6NQ9QZ9N)",
            "gatekeeper-assess": false,
            "entitlements": "frontend/entitlements.plist",
            "entitlements-inherit": "frontend/entitlements.plist"
        },
        "osxNotarize": {
            "appleId": "tech@quorumcontrol.com",
            "appleIdPassword": "@keychain:jasons-game-notarization-password"
        }
    },
    "electronWinstallerConfig": {
        "name": "jasons_game"
    },
    "electronInstallerDebian": {},
    "electronInstallerRedhat": {},
    "github_repository": {
        "owner": "quorumcontrol",
        "name": "jasons-game"
    },
    "windowsStoreConfig": {
        "packageName": "",
        "name": "JasonsGame"
    },
    "hooks": {
        "packageAfterPrune": deleteOtherPlatforms
    },
    "publishers": [
        {
            "name": "@electron-forge/publisher-github",
            "config": {
                "repository": {
                    "owner": "quorumcontrol",
                    "name": "jasons-game"
                },
                "prerelease": true
            }
        }
    ]
};
