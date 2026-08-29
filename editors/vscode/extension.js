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
  const workspaceFolder = vscodeApi.workspace.getWorkspaceFolder(document.uri);

  // A task terminal may only start in the user's home directory when no folder
  // is open, which would run a standalone file from the wrong directory or fail
  // outright. Such a file runs in its own terminal instead, still launched from
  // an argument vector rather than a shell command string.
  if (selectRunStrategy(workspaceFolder) === STANDALONE_STRATEGY) {
    runStandalone(vscodeApi, executable, filePath);
    return;
  }

  const taskName = `AhdCode: Run ${path.basename(filePath)}`;
  const execution = new vscodeApi.ProcessExecution(
    executable,
    ["run", filePath],
    { cwd: path.dirname(filePath) },
  );
  const scope = workspaceFolder || vscodeApi.TaskScope.Global;
  const task = new vscodeApi.Task(
    { type: "ahdcode", task: "runFile" },
    scope,
    taskName,
    "AhdCode",
    execution,
    [],
  );

  // One dedicated terminal is reused across runs, and it is never cleared, so
  // each run appends below the previous output instead of erasing it.
  task.presentationOptions = {
    reveal: vscodeApi.TaskRevealKind.Always,
    echo: true,
    focus: false,
    panel: vscodeApi.TaskPanelKind.Dedicated,
    showReuseMessage: false,
    clear: false,
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

const TASK_STRATEGY = "task";
const STANDALONE_STRATEGY = "standalone";
const STANDALONE_TERMINAL_NAME = "AhdCode";

// selectRunStrategy decides how one run is launched. A document inside an open
// folder keeps the dedicated task terminal; a standalone file does not, because
// the host restricts a task terminal's working directory without a workspace.
function selectRunStrategy(workspaceFolder) {
  return workspaceFolder ? TASK_STRATEGY : STANDALONE_STRATEGY;
}

// standaloneTerminalOptions launches the compiler as the terminal's own
// process. shellPath and shellArgs are an argument vector, so a path containing
// spaces, quotes, $, ;, &, parentheses, or Unicode is never interpreted by a
// shell, and cwd stays the source file's directory so relative paths in the
// program still resolve the way they do on the command line.
function standaloneTerminalOptions(executable, filePath) {
  return {
    name: STANDALONE_TERMINAL_NAME,
    cwd: path.dirname(filePath),
    shellPath: executable,
    shellArgs: ["run", filePath],
  };
}

// standaloneTerminal is the one terminal this fallback owns. Its process ends
// when the program ends, so a rerun replaces it rather than reusing a dead
// shell; only one AhdCode terminal is ever left behind.
let standaloneTerminal;

function runStandalone(vscodeApi, executable, filePath) {
  if (standaloneTerminal) {
    standaloneTerminal.dispose();
    standaloneTerminal = undefined;
  }
  standaloneTerminal = vscodeApi.window.createTerminal(
    standaloneTerminalOptions(executable, filePath),
  );
  standaloneTerminal.show(true);
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
  selectRunStrategy,
  standaloneTerminalOptions,
  TASK_STRATEGY,
  STANDALONE_STRATEGY,
  deactivate,
  runFile,
  findExecutable,
  isRunnableDocument,
  COMMAND_ID,
  INVALID_EDITOR_MESSAGE,
  MISSING_EXECUTABLE_MESSAGE,
  SAVE_FAILED_MESSAGE,
};
