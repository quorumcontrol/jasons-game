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
        /\/frontend\/main\.js$/
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
            name: '@electron-forge/maker-zip',
            platforms: ['darwin', 'linux']
        },
        {
            name: '@electron-forge/maker-squirrel'
        }
    ],
    "packagerConfig": {
        "packageManager": "npm",
        "asar": {
            "unpackDir": 'bin'
        },
        "ignore": ignorePath
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
    }
};