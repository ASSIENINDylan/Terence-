// Point d'entrée Electron : ouvre le jeu (index.html) dans une fenêtre d'application.
const { app, BrowserWindow, Menu } = require("electron");
const path = require("path");

function createWindow() {
  const win = new BrowserWindow({
    width: 664,
    height: 560,
    resizable: true,
    backgroundColor: "#0b0d12",
    title: "Shadow of the Warrior",
    icon: path.join(__dirname, "build", "icon.ico"),
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  // Pas de barre de menu : c'est un jeu.
  Menu.setApplicationMenu(null);

  win.loadFile("index.html");

  // F11 pour basculer en plein écran.
  win.webContents.on("before-input-event", (event, input) => {
    if (input.type === "keyDown" && input.key === "F11") {
      win.setFullScreen(!win.isFullScreen());
    }
  });
}

app.whenReady().then(() => {
  createWindow();
  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});
