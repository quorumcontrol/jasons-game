const { ipcRenderer, remote } = require('electron');

const _ipcRenderer = ipcRenderer;
const _appVersion = remote.app.getVersion();

process.once('loaded', () => {
    global.autoUpdaterQuitAndInstall = () => {
        _ipcRenderer.send('auto-updater', 'quitAndInstall');
    };

    global.appVersion = _appVersion;
});
