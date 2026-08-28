"use strict";

const assert = require("node:assert/strict");
const path = require("node:path");
const vscode = require("vscode");

const executable = process.env.AHDCODE_TEST_EXECUTABLE;

async function run() {
	assert.ok(executable, "AHDCODE_TEST_EXECUTABLE must name the smoke-test compiler");
  const developmentExtension = vscode.extensions.getExtension("ahdcode-local.ahdcode");
  assert.ok(developmentExtension, "development extension is discoverable");
  await developmentExtension.activate();

  await vscode.workspace
    .getConfiguration("ahdcode")
    .update("executablePath", executable, vscode.ConfigurationTarget.Global);

  const commands = await vscode.commands.getCommands(true);
  assert.ok(commands.includes("ahdcode.runFile"), "run command is registered");

  const cases = [
    { file: "/tmp/test.ahd", exitCode: 0, makeDirty: true },
    { file: "/tmp/Ahd Code Tests/test file.ahd", exitCode: 0 },
    { file: "/tmp/öğrenci/deneme.ahd", exitCode: 0 },
    { file: "/tmp/ahdcode-editor-smoke/compile-error.ahd", nonzero: true },
    { file: "/tmp/ahdcode-editor-smoke/runtime-error.ahd", nonzero: true },
  ];

  for (const testCase of cases) {
    const result = await runAhdCodeFile(testCase.file, testCase.makeDirty);
    // Some Electron task hosts report an unknown process exit code even though
    // the visible terminal task completed. Validate it when the host exposes it;
    // the CLI fixtures are also checked directly by the outer smoke command.
    if (result.exitCode !== undefined) {
      if (testCase.nonzero) {
        assert.notEqual(result.exitCode, 0, `${testCase.file} must fail visibly`);
      } else {
        assert.equal(result.exitCode, testCase.exitCode, `${testCase.file} must run`);
      }
    }
    assert.equal(result.taskName, `AhdCode: Run ${path.basename(testCase.file)}`);
    assert.equal(result.definition.type, "ahdcode");
    assert.equal(result.definition.task, "runFile");
    assert.equal(result.presentationOptions.clear, true);
    assert.equal(result.runOptions.instanceLimit, 1);
    console.log(`AhdCode task completed: ${testCase.file} (exit=${result.exitCode})`);
  }

  const missingTaskStarted = await runWithMissingExecutable();
  assert.equal(missingTaskStarted, false, "missing executable must not start a task");
}

async function runAhdCodeFile(filePath, makeDirty = false) {
  const document = await vscode.workspace.openTextDocument(vscode.Uri.file(filePath));
  await vscode.window.showTextDocument(document);
  assert.equal(document.languageId, "ahdcode", `${filePath} language association`);

  if (makeDirty) {
    const edit = new vscode.WorkspaceEdit();
    edit.insert(document.uri, new vscode.Position(document.lineCount, 0), "\n// auto-save smoke");
    assert.equal(await vscode.workspace.applyEdit(edit), true);
    assert.equal(document.isDirty, true, "fixture becomes dirty before run");
  }

  const expectedName = `AhdCode: Run ${path.basename(filePath)}`;
  const completion = waitForTask(expectedName, document);
  await vscode.commands.executeCommand("ahdcode.runFile");
  const result = await completion;
  assert.equal(document.isDirty, false, "run command saves before task starts");
  return result;
}

function waitForTask(expectedName, document) {
  return new Promise((resolve, reject) => {
    let startedTask;
    let dirtyAtStart;
    const timeout = setTimeout(() => {
      dispose();
      reject(new Error(`timed out waiting for ${expectedName}`));
    }, 120000);

    const startDisposable = vscode.tasks.onDidStartTask((event) => {
      if (event.execution.task.name === expectedName) {
        startedTask = event.execution.task;
        dirtyAtStart = document.isDirty;
      }
    });
    const endDisposable = vscode.tasks.onDidEndTaskProcess((event) => {
      if (event.execution.task.name !== expectedName) {
        return;
      }
      const task = startedTask || event.execution.task;
      dispose();
      assert.equal(dirtyAtStart, false, "document was clean when task started");
      resolve({
        exitCode: event.exitCode,
        taskName: task.name,
        definition: task.definition,
        presentationOptions: task.presentationOptions,
        runOptions: task.runOptions,
      });
    });

    function dispose() {
      clearTimeout(timeout);
      startDisposable.dispose();
      endDisposable.dispose();
    }
  });
}

async function runWithMissingExecutable() {
  await vscode.workspace
    .getConfiguration("ahdcode")
    .update(
      "executablePath",
      path.join(path.dirname(executable), "does-not-exist"),
      vscode.ConfigurationTarget.Global,
    );

  const document = await vscode.workspace.openTextDocument(vscode.Uri.file("/tmp/test.ahd"));
  await vscode.window.showTextDocument(document);
  let started = false;
  const disposable = vscode.tasks.onDidStartTask((event) => {
    if (event.execution.task.definition.type === "ahdcode") {
      started = true;
    }
  });

  await vscode.commands.executeCommand("ahdcode.runFile");
  await new Promise((resolve) => setTimeout(resolve, 300));
  disposable.dispose();

  await vscode.workspace
    .getConfiguration("ahdcode")
    .update("executablePath", executable, vscode.ConfigurationTarget.Global);
  return started;
}

module.exports = { run };
