"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const Module = require("node:module");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const state = {
  activeTextEditor: undefined,
  configurationPath: "",
  errors: [],
  tasks: [],
  terminals: [],
  workspaceFolder: { name: "workspace" },
  handlers: new Map(),
};

class ProcessExecution {
  constructor(process, args, options) {
    this.process = process;
    this.args = args;
    this.options = options;
  }
}

class Task {
  constructor(definition, scope, name, source, execution, problemMatchers) {
    Object.assign(this, {
      definition,
      scope,
      name,
      source,
      execution,
      problemMatchers,
    });
  }
}

const vscodeMock = {
  commands: {
    registerCommand(command, handler) {
      state.handlers.set(command, handler);
      return { dispose() {} };
    },
  },
  window: {
    get activeTextEditor() {
      return state.activeTextEditor;
    },
    async showErrorMessage(message) {
      state.errors.push(message);
    },
    createTerminal(options) {
      const terminal = {
        options,
        disposed: false,
        shown: 0,
        dispose() {
          this.disposed = true;
        },
        show() {
          this.shown += 1;
        },
      };
      state.terminals.push(terminal);
      return terminal;
    },
  },
  workspace: {
    getConfiguration() {
      return {
        get(_name, defaultValue) {
          return state.configurationPath || defaultValue;
        },
      };
    },
    getWorkspaceFolder() {
      return state.workspaceFolder;
    },
  },
  tasks: {
    async executeTask(task) {
      state.tasks.push(task);
      return { task };
    },
  },
  ProcessExecution,
  Task,
  TaskScope: { Global: "global" },
  TaskRevealKind: { Always: "always" },
  TaskPanelKind: { Dedicated: "dedicated" },
};

const originalLoad = Module._load;
Module._load = function load(request, parent, isMain) {
  if (request === "vscode") {
    return vscodeMock;
  }
  return originalLoad.call(this, request, parent, isMain);
};
const extension = require("../extension");
Module._load = originalLoad;

function reset() {
  state.activeTextEditor = undefined;
  state.configurationPath = "";
  state.errors.length = 0;
  state.tasks.length = 0;
  state.terminals.length = 0;
  state.workspaceFolder = { name: "workspace" };
  state.handlers.clear();
}

function documentFor(filePath, options = {}) {
  return {
    isUntitled: false,
    isDirty: false,
    uri: { scheme: "file", fsPath: filePath },
    async save() {
      if (options.throwOnSave) {
        throw new Error("save failed");
      }
      return options.saveResult !== false;
    },
  };
}

function makeExecutable(directory) {
  const executable = path.join(directory, "ahdcode test executable");
  fs.writeFileSync(executable, "#!/bin/sh\nexit 0\n", { mode: 0o755 });
  return executable;
}

test.beforeEach(reset);

test("activate registers ahdcode.runFile", () => {
  const subscriptions = [];
  extension.activate({ subscriptions });
  assert.equal(subscriptions.length, 1);
  assert.equal(typeof state.handlers.get("ahdcode.runFile"), "function");
});

test("rejects a missing, untitled, or non-AhdCode editor", async () => {
  await extension.runFile(vscodeMock, { env: { PATH: "" } });
  assert.deepEqual(state.errors, [extension.INVALID_EDITOR_MESSAGE]);

  reset();
  const document = documentFor("/tmp/test.txt");
  state.activeTextEditor = { document };
  await extension.runFile(vscodeMock, { env: { PATH: "" } });
  assert.deepEqual(state.errors, [extension.INVALID_EDITOR_MESSAGE]);
  assert.equal(state.tasks.length, 0);
});

test("save failure prevents stale execution", async () => {
  const document = documentFor("/tmp/test.ahd", { saveResult: false });
  state.activeTextEditor = { document };

  await extension.runFile(vscodeMock, { env: { PATH: "" } });

  assert.deepEqual(state.errors, [extension.SAVE_FAILED_MESSAGE]);
  assert.equal(state.tasks.length, 0);
});

test("missing executable reports the required error without a cascade", async () => {
  state.activeTextEditor = { document: documentFor("/tmp/test.ahd") };

  await extension.runFile(vscodeMock, { env: { PATH: "" } });

  assert.deepEqual(state.errors, [extension.MISSING_EXECUTABLE_MESSAGE]);
  assert.equal(state.tasks.length, 0);
});

test("uses ProcessExecution argument arrays for spaces and Unicode paths", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-extension-"));
  const executable = makeExecutable(temporary);
  const filePath = "/tmp/Ahd Code Tests/öğrenci deneme.ahd";
  state.configurationPath = executable;
  state.activeTextEditor = { document: documentFor(filePath) };

  await extension.runFile(vscodeMock);

  assert.equal(state.errors.length, 0);
  assert.equal(state.tasks.length, 1);
  const task = state.tasks[0];
  assert.equal(task.name, "AhdCode: Run öğrenci deneme.ahd");
  assert.equal(task.execution.process, executable);
  assert.deepEqual(task.execution.args, ["run", filePath]);
  assert.deepEqual(task.execution.options, { cwd: "/tmp/Ahd Code Tests" });
  assert.equal(task.presentationOptions.panel, "dedicated");
  assert.equal(task.presentationOptions.clear, false);
  assert.equal(task.runOptions.instanceLimit, 1);

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("repeated runs append to one terminal instead of clearing it", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-rerun-"));
  const executable = makeExecutable(temporary);
  state.configurationPath = executable;
  state.activeTextEditor = { document: documentFor("/tmp/loop.ahd") };

  await extension.runFile(vscodeMock);
  await extension.runFile(vscodeMock);
  await extension.runFile(vscodeMock);

  assert.equal(state.errors.length, 0);
  assert.equal(state.tasks.length, 3);
  for (const task of state.tasks) {
    // Every run targets the same dedicated panel, so no run opens a new
    // terminal, and none of them erases what the previous run printed.
    assert.equal(task.presentationOptions.panel, "dedicated");
    assert.equal(task.presentationOptions.clear, false);
    assert.equal(task.presentationOptions.reveal, "always");
    assert.equal(task.presentationOptions.showReuseMessage, false);
    assert.equal(task.runOptions.instanceLimit, 1);
    assert.equal(task.name, "AhdCode: Run loop.ahd");
    // The dedicated panel is keyed on task identity, so every run must present
    // the same definition, type, source, and scope. A run that varied any of
    // them would be a different task and would get its own terminal.
    assert.deepEqual(task.definition, { type: "ahdcode", task: "runFile" });
    assert.equal(task.source, "AhdCode");
    assert.equal(task.scope, state.workspaceFolder);
    // The program is launched directly, never through a shell command string,
    // so an interactive take() reads the terminal's own stdin.
    assert.equal(task.execution instanceof ProcessExecution, true);
    assert.equal(Array.isArray(task.execution.args), true);
  }
  const identities = new Set(
    state.tasks.map((task) =>
      JSON.stringify([task.definition, task.source, task.name]),
    ),
  );
  assert.equal(identities.size, 1, "repeated runs must present one task identity");

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("a run of a different file still reuses the dedicated panel", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-switch-"));
  const executable = makeExecutable(temporary);
  state.configurationPath = executable;

  state.activeTextEditor = { document: documentFor("/tmp/first.ahd") };
  await extension.runFile(vscodeMock);
  state.activeTextEditor = { document: documentFor("/tmp/second.ahd") };
  await extension.runFile(vscodeMock);

  assert.equal(state.tasks.length, 2);
  assert.equal(state.tasks[0].presentationOptions.clear, false);
  assert.equal(state.tasks[1].presentationOptions.clear, false);
  assert.equal(state.tasks[1].execution.args[1], "/tmp/second.ahd");
  // Only the display name follows the file; the identity that selects the
  // dedicated terminal does not, so both files share one terminal.
  assert.deepEqual(state.tasks[0].definition, state.tasks[1].definition);
  assert.equal(state.tasks[0].source, state.tasks[1].source);
  assert.equal(state.tasks[0].scope, state.tasks[1].scope);
  assert.equal(state.tasks[0].execution.options.cwd, "/tmp");
  assert.equal(state.tasks[1].execution.options.cwd, "/tmp");

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("a run never builds a shell command string", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-argv-"));
  const executable = makeExecutable(temporary);
  state.configurationPath = executable;
  // A path carrying shell metacharacters must survive as one argv element.
  const filePath = "/tmp/ahd tests/$(touch pwned); rm -rf * & 'quoted' \"x\".ahd";
  state.activeTextEditor = { document: documentFor(filePath) };

  await extension.runFile(vscodeMock);

  assert.equal(state.errors.length, 0);
  const task = state.tasks[0];
  assert.equal(task.execution instanceof ProcessExecution, true);
  assert.equal(task.execution.process, executable);
  assert.deepEqual(task.execution.args, ["run", filePath]);
  assert.equal(task.execution.options.cwd, "/tmp/ahd tests");
  assert.equal(typeof task.execution.commandLine, "undefined");

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("a standalone file with no workspace runs in its own terminal", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-standalone-"));
  const executable = makeExecutable(temporary);
  state.configurationPath = executable;
  state.workspaceFolder = undefined;
  state.activeTextEditor = { document: documentFor("/tmp/Ahd Code/öğrenci deneme.ahd") };

  await extension.runFile(vscodeMock);

  // A task terminal cannot choose this working directory without a workspace,
  // so no task is created at all.
  assert.equal(state.errors.length, 0);
  assert.equal(state.tasks.length, 0);
  assert.equal(state.terminals.length, 1);
  const terminal = state.terminals[0];
  assert.equal(terminal.options.name, "AhdCode");
  assert.equal(terminal.options.cwd, "/tmp/Ahd Code");
  assert.equal(terminal.options.shellPath, executable);
  assert.deepEqual(terminal.options.shellArgs, ["run", "/tmp/Ahd Code/öğrenci deneme.ahd"]);
  assert.equal(terminal.shown, 1);

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("a standalone rerun replaces its terminal instead of accumulating", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-standalone-rerun-"));
  const executable = makeExecutable(temporary);
  state.configurationPath = executable;
  state.workspaceFolder = undefined;
  state.activeTextEditor = { document: documentFor("/tmp/loop.ahd") };

  await extension.runFile(vscodeMock);
  await extension.runFile(vscodeMock);

  assert.equal(state.terminals.length, 2);
  // The first terminal's process has already ended, so it is disposed rather
  // than left behind as a dead shell.
  assert.equal(state.terminals[0].disposed, true);
  assert.equal(state.terminals[1].disposed, false);

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("the standalone fallback never builds a shell command string", () => {
  const filePath = "/tmp/ahd tests/$(touch pwned); rm -rf * & 'quoted' \"x\".ahd";
  const options = extension.standaloneTerminalOptions("/usr/local/bin/ahdcode", filePath);
  assert.equal(options.shellPath, "/usr/local/bin/ahdcode");
  assert.deepEqual(options.shellArgs, ["run", filePath]);
  assert.equal(options.cwd, "/tmp/ahd tests");
  // shellArgs is an argument vector; there is no command line to interpret.
  assert.equal(typeof options.commandLine, "undefined");
  assert.equal(path.isAbsolute(options.shellArgs[1]), true);
});

test("the run strategy follows the workspace, not the file", () => {
  assert.equal(extension.selectRunStrategy({ name: "folder" }), extension.TASK_STRATEGY);
  assert.equal(extension.selectRunStrategy(undefined), extension.STANDALONE_STRATEGY);
});

test("findExecutable resolves ahdcode from PATH", async () => {
  const temporary = fs.mkdtempSync(path.join(os.tmpdir(), "ahdcode-path-"));
  const executable = path.join(temporary, "ahdcode");
  fs.writeFileSync(executable, "#!/bin/sh\nexit 0\n", { mode: 0o755 });

  assert.equal(
    await extension.findExecutable("ahdcode", { PATH: temporary }, process.platform),
    executable,
  );

  fs.rmSync(temporary, { recursive: true, force: true });
});

test("manifest exposes the portable command, menu, keybinding, and language", () => {
  const manifest = require("../package.json");
  assert.equal(manifest.main, "./extension.js");
  assert.equal(manifest.version, "0.1.3");
  assert.equal(manifest.icon, "images/ahdcode-icon.png");
  assert.equal(manifest.engines.vscode, "^1.107.0");
  assert.equal(manifest.contributes.commands[0].command, "ahdcode.runFile");
  assert.equal(manifest.contributes.commands[0].icon, "$(play)");
  assert.equal(manifest.contributes.menus["editor/title"][0].when, "resourceLangId == ahdcode");
  assert.equal(manifest.contributes.keybindings[0].key, "f6");
  assert.equal(manifest.contributes.keybindings[0].when, "editorTextFocus && editorLangId == ahdcode");
  assert.deepEqual(manifest.contributes.languages[0].extensions, [".ahd"]);
  assert.deepEqual(manifest.contributes.languages[0].icon, {
    light: "./icons/ahdcode-file-light.png",
    dark: "./icons/ahdcode-file-dark.png",
  });
  for (const relative of [
    manifest.icon,
    manifest.contributes.languages[0].icon.light,
    manifest.contributes.languages[0].icon.dark,
  ]) {
    assert.equal(fs.existsSync(path.join(__dirname, "..", relative)), true);
  }
});

test("TextMate grammar follows the frozen v0.1 lexical surface", () => {
  const grammar = require("../syntaxes/ahdcode.tmLanguage.json");
  const numberPatterns = grammar.repository.numbers.patterns.map(
    ({ match }) => new RegExp(match, "u"),
  );
  const matchesNumber = (text) => numberPatterns.some((pattern) => pattern.test(text));

  assert.equal(matchesNumber("123"), true);
  assert.equal(matchesNumber("1.25"), true);
  assert.equal(matchesNumber("1e3"), true);
  assert.equal(matchesNumber("0x10"), false);
  assert.equal(matchesNumber("1_000"), false);

  const escape = new RegExp(
    grammar.repository.strings.patterns[0].patterns[0].match,
    "u",
  );
  assert.equal(escape.test("\\n"), true);
  assert.equal(escape.test("\\{"), true);
  assert.equal(escape.test("\\0"), false);
  assert.equal(escape.test("\\u{41}"), false);

  const grammarText = JSON.stringify(grammar);
  for (const word of ["default", "same", "is", "has", "lambda", "structure", "attribute", "SuperClass"]) {
    assert.equal(grammarText.includes(word), true);
  }
});
