"use strict";

const fs = require("node:fs");
const path = require("node:path");
const vscode = require("vscode");

const COMMAND_ID = "ahdcode.runFile";
const MISSING_EXECUTABLE_MESSAGE =
  "AhdCode executable was not found on PATH.\n" +
  "Install AhdCode or add its binary directory to PATH.";
const SAVE_FAILED_MESSAGE =
  "AhdCode could not save the active file. The program was not run.";
const INVALID_EDITOR_MESSAGE =
  "Open a saved .ahd file before running AhdCode.";

function activate(context) {
  const disposable = vscode.commands.registerCommand(COMMAND_ID, () => runFile());
  context.subscriptions.push(disposable);
}

async function runFile(vscodeApi = vscode, runtime = {}) {
  const editor = vscodeApi.window.activeTextEditor;
  const document = editor && editor.document;

  if (!isRunnableDocument(document)) {
    void vscodeApi.window.showErrorMessage(INVALID_EDITOR_MESSAGE);
    return;
  }

  let saved;
  try {
    saved = await document.save();
  } catch (_error) {
    saved = false;
  }

  if (!saved || document.isDirty) {
    void vscodeApi.window.showErrorMessage(SAVE_FAILED_MESSAGE);
    return;
  }

  const configuredPath = vscodeApi.workspace
    .getConfiguration("ahdcode")
    .get("executablePath", "")
    .trim();
  const executable = await findExecutable(
    configuredPath || "ahdcode",
    runtime.env || process.env,
    runtime.platform || process.platform,
  );

  if (!executable) {
    void vscodeApi.window.showErrorMessage(MISSING_EXECUTABLE_MESSAGE);
    return;
  }

  const filePath = document.uri.fsPath;
  const taskName = `AhdCode: Run ${path.basename(filePath)}`;
  const execution = new vscodeApi.ProcessExecution(
    executable,
    ["run", filePath],
    { cwd: path.dirname(filePath) },
  );
  const workspaceFolder = vscodeApi.workspace.getWorkspaceFolder(document.uri);
  const scope = workspaceFolder || vscodeApi.TaskScope.Global;
  const task = new vscodeApi.Task(
    { type: "ahdcode", task: "runFile" },
    scope,
    taskName,
    "AhdCode",
    execution,
    [],
  );

  task.presentationOptions = {
    reveal: vscodeApi.TaskRevealKind.Always,
    echo: true,
    focus: false,
    panel: vscodeApi.TaskPanelKind.Dedicated,
    showReuseMessage: false,
    clear: true,
  };
  task.runOptions = {
    instanceLimit: 1,
    reevaluateOnRerun: true,
  };

  try {
    await vscodeApi.tasks.executeTask(task);
  } catch (error) {
    const detail = error instanceof Error ? ` ${error.message}` : "";
    void vscodeApi.window.showErrorMessage(
      `AhdCode could not start the run task.${detail}`,
    );
  }
}

function isRunnableDocument(document) {
  return Boolean(
    document &&
      !document.isUntitled &&
      document.uri &&
      document.uri.scheme === "file" &&
      path.extname(document.uri.fsPath).toLowerCase() === ".ahd",
  );
}

async function findExecutable(command, environment, platform) {
  const value = command.trim();
  if (!value) {
    return undefined;
  }

  if (path.isAbsolute(value) || value.includes("/") || value.includes("\\")) {
    const candidate = path.resolve(value);
    return (await isExecutableFile(candidate, platform)) ? candidate : undefined;
  }

  const pathValue = environment.PATH || environment.Path || environment.path || "";
  const directories = pathValue.split(path.delimiter);
  const extensions = executableExtensions(value, environment, platform);

  for (const directory of directories) {
    if (!directory) {
      continue;
    }
    for (const extension of extensions) {
      const candidate = path.join(directory, value + extension);
      if (await isExecutableFile(candidate, platform)) {
        return candidate;
      }
    }
  }

  return undefined;
}

function executableExtensions(command, environment, platform) {
  if (platform !== "win32" || path.extname(command)) {
    return [""];
  }

  const pathExt = environment.PATHEXT || ".COM;.EXE;.BAT;.CMD";
  return pathExt
    .split(";")
    .filter(Boolean)
    .map((extension) => extension.toLowerCase());
}

async function isExecutableFile(candidate, platform) {
  try {
    const mode = platform === "win32" ? fs.constants.F_OK : fs.constants.X_OK;
    await fs.promises.access(candidate, mode);
    return (await fs.promises.stat(candidate)).isFile();
  } catch (_error) {
    return false;
  }
}

function deactivate() {}

module.exports = {
  activate,
  deactivate,
  runFile,
  findExecutable,
  isRunnableDocument,
  COMMAND_ID,
  INVALID_EDITOR_MESSAGE,
  MISSING_EXECUTABLE_MESSAGE,
  SAVE_FAILED_MESSAGE,
};
