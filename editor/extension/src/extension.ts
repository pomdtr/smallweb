'use strict';

import * as vscode from 'vscode';
import { SmallwebFS } from './fileSystemProvider';

export function activate(context: vscode.ExtensionContext) {
	const smallwebFS = new SmallwebFS();
	context.subscriptions.push(vscode.workspace.registerFileSystemProvider('smallweb', smallwebFS, { isCaseSensitive: true }));
	vscode.workspace.registerFileSearchProvider("smallweb", smallwebFS)
	vscode.workspace.registerTextSearchProvider("smallweb", smallwebFS)
}
