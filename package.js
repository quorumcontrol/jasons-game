const packager = require('electron-packager');
const fs = require('fs');
const path = require('path');

function ignorePath(path) {
    let includedPaths = [
        /\/package(?:-lock)?\.json$/,
        /\/bin$/,
        /\/bin\//,
        /\/main\.js$/
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

function deleteOtherPlatforms(buildPath, electronVersion, platform, arch, callback) {
    let ourBinary = `jasonsgame-${platform}-public`;

    if (platform === "win32") {
        ourBinary += '.exe';
    }

    let binPath = path.join(buildPath, 'bin');

    fs.readdir(binPath, (err, binaries) => {
       if (err !== null) {
           console.log(`Error getting game binaries: ${err}`);
           callback();
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

    callback();
}

async function bundleElectronApp(options) {
    const appPaths = await packager(options);
    console.log(`Electron app bundles created:\n${appPaths.join("\n")}`);
}

(async () => {
    let options = {
        "dir": ".",
        "platform": ["darwin", "win32", "linux"],
        "arch": "x64",
        "out": "dist",
        "overwrite": true,
        "ignore": ignorePath,
        "afterPrune": [deleteOtherPlatforms]
    };

    await bundleElectronApp(options);
})();
