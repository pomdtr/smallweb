/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/


import * as vscode from 'vscode';
import * as path from "path-browserify"
import { XMLParser } from "fast-xml-parser"


export class SmallwebFS implements vscode.FileSystemProvider, vscode.FileSearchProvider, vscode.TextSearchProvider {
	#parser = new XMLParser()

	async fetch(uri: vscode.Uri, init: RequestInit): Promise<Response> {
		return await fetch(`/webdav${uri.path}`, {
			...init,
		})
	}

	// --- manage file metadata
	async stat(uri: vscode.Uri): Promise<vscode.FileStat> {
		const resp = await this.fetch(uri, { method: "PROPFIND", headers: { "Depth": "0" } })
		if (!resp.ok) {
			throw vscode.FileSystemError.FileNotFound(uri)
		}

		const xml = await resp.text()
		const data = this.#parser.parse(xml)

		const stat = data["D:multistatus"]["D:response"]["D:propstat"]["D:prop"]
		const type = stat["D:resourcetype"] !== "" ? vscode.FileType.Directory : vscode.FileType.File
		const size = stat["D:getcontentlength"] || 0
		const mtime = new Date(stat["D:getlastmodified"]).getTime()

		return {
			type,
			size,
			mtime,
			ctime: 0,
		}
	}

	async readDirectory(uri: vscode.Uri): Promise<[string, vscode.FileType][]> {
		const resp = await this.fetch(uri, { method: "PROPFIND", headers: { "Depth": "1" } })
		if (!resp.ok) {
			throw vscode.FileSystemError.FileNotFound(uri)
		}

		const xml = await resp.text()
		const data = this.#parser.parse(xml)

		const entries = data["D:multistatus"]["D:response"]
		return entries.slice(1).map((entry: any) => {
			const stat = entry["D:propstat"]["D:prop"]
			const name = path.posix.basename(entry["D:href"])
			const type = stat["D:resourcetype"] !== "" ? vscode.FileType.Directory : vscode.FileType.File
			return [decodeURIComponent(name), type]
		})
	}

	// --- manage file contents

	async readFile(uri: vscode.Uri): Promise<Uint8Array> {
		const resp = await this.fetch(uri, { method: "GET" })
		if (!resp.ok) {
			throw vscode.FileSystemError.FileNotFound(uri)
		}

		return new Uint8Array(await resp.arrayBuffer())
	}

	async writeFile(uri: vscode.Uri, content: Uint8Array, options: { create: boolean, overwrite: boolean }): Promise<void> {
		try {
			this.stat(uri)
			const resp = await this.fetch(uri, { method: "PUT", body: content })
			if (!resp.ok) {
				throw vscode.FileSystemError.FileNotFound(uri)
			}

			this._fireSoon({ type: vscode.FileChangeType.Changed, uri });
		} catch (e) {
			const resp = await this.fetch(uri, { method: "PUT", body: content })
			if (!resp.ok) {
				throw vscode.FileSystemError.FileNotFound(uri)
			}

			this._fireSoon({ type: vscode.FileChangeType.Created, uri });
		}
	}


	async rename(oldUri: vscode.Uri, newUri: vscode.Uri, options: { overwrite: boolean }): Promise<void> {
		const resp = await this.fetch(oldUri, {
			method: "MOVE", headers: {
				"Destination": "/webdav" + newUri.path,
				"Overwrite": options.overwrite ? "T" : "F"
			}
		})
		if (!resp.ok) {
			switch (resp.status) {
				case 404:
					throw vscode.FileSystemError.FileNotFound(oldUri)
				case 403:
					throw vscode.FileSystemError.NoPermissions(oldUri)
				default:
					throw vscode.FileSystemError.Unavailable(oldUri)
			}
		}

		this._fireSoon(
			{ type: vscode.FileChangeType.Deleted, uri: oldUri },
			{ type: vscode.FileChangeType.Created, uri: newUri }
		);
	}

	async copy(source: vscode.Uri, destination: vscode.Uri, options: { readonly overwrite: boolean; }) {
		const resp = await this.fetch(source, {
			method: "COPY", headers: {
				"Destination": "/webdav" + destination.path,
				"Overwrite": options.overwrite ? "T" : "F"
			}
		})
		if (!resp.ok) {
			throw vscode.FileSystemError.FileNotFound(source)
		}
		this._fireSoon({ type: vscode.FileChangeType.Created, uri: destination });
	}

	async delete(uri: vscode.Uri): Promise<void> {
		const resp = await this.fetch(uri, { method: "DELETE" })
		if (!resp.ok) {
			throw vscode.FileSystemError.FileNotFound(uri)
		}

		const parent = uri.with({ path: path.posix.dirname(uri.path) })
		this._fireSoon({ type: vscode.FileChangeType.Changed, uri: parent }, { uri, type: vscode.FileChangeType.Deleted });
	}

	async createDirectory(uri: vscode.Uri): Promise<void> {
		const resp = await this.fetch(uri, { method: "MKCOL" })
		if (!resp.ok) {
			throw vscode.FileSystemError.FileNotFound(uri)
		}

		const parent = uri.with({ path: path.posix.dirname(uri.path) })
		this._fireSoon({ type: vscode.FileChangeType.Created, uri }, { type: vscode.FileChangeType.Changed, uri: parent });
	}

	async provideFileSearchResults(query: vscode.FileSearchQuery, options: vscode.FileSearchOptions, token: vscode.CancellationToken): Promise<vscode.Uri[]> {
		return []
	}

	async provideTextSearchResults(query: vscode.TextSearchQuery, options: vscode.TextSearchOptions, progress: vscode.Progress<vscode.TextSearchResult>, token: vscode.CancellationToken): Promise<vscode.TextSearchComplete> {
		return {
			limitHit: true,
			message: []
		}
	}


	// --- manage file events

	private _emitter = new vscode.EventEmitter<vscode.FileChangeEvent[]>();
	private _bufferedEvents: vscode.FileChangeEvent[] = [];
	private _fireSoonHandle?: NodeJS.Timeout;

	readonly onDidChangeFile: vscode.Event<vscode.FileChangeEvent[]> = this._emitter.event;

	watch(_resource: vscode.Uri): vscode.Disposable {
		// ignore, fires for all changes...
		return new vscode.Disposable(() => { });
	}

	private _fireSoon(...events: vscode.FileChangeEvent[]): void {
		this._bufferedEvents.push(...events);

		if (this._fireSoonHandle) {
			clearTimeout(this._fireSoonHandle);
		}

		this._fireSoonHandle = setTimeout(() => {
			this._emitter.fire(this._bufferedEvents);
			this._bufferedEvents.length = 0;
		}, 5);
	}
}
