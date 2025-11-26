/**
 * CodeIndex & Search Module
 *
 * Provides full-text search (BM25-like) and regex search capabilities
 * for generated TypeScript tool wrappers.
 */

import type {
	GeneratedToolInfo,
	SearchQuery,
	SearchResult,
	SearchResultItem,
} from "../types.js";

/**
 * Simple BM25-based scoring for text search
 */
class BM25Index {
	private documents: Map<string, GeneratedToolInfo> = new Map();
	private terms: Map<string, Set<string>> = new Map(); // term -> document ids
	private docTermFreq: Map<string, Map<string, number>> = new Map(); // docId -> term -> freq
	private avgDocLength = 0;
	private k1 = 1.5;
	private b = 0.75;

	/**
	 * Tokenize text into searchable terms
	 */
	private tokenize(text: string): string[] {
		return text
			.toLowerCase()
			.replace(/[^a-z0-9_]/g, " ")
			.split(/\s+/)
			.filter((t) => t.length > 1);
	}

	/**
	 * Get searchable text from a document
	 */
	private getSearchableText(doc: GeneratedToolInfo): string {
		return [
			doc.toolName,
			doc.functionName,
			doc.description || "",
			doc.sourceCode,
			doc.serverId,
		].join(" ");
	}

	/**
	 * Add a document to the index
	 */
	add(doc: GeneratedToolInfo): void {
		const docId = `${doc.serverId}.${doc.toolName}`;
		this.documents.set(docId, doc);

		const text = this.getSearchableText(doc);
		const tokens = this.tokenize(text);

		// Count term frequencies
		const termFreq = new Map<string, number>();
		for (const term of tokens) {
			termFreq.set(term, (termFreq.get(term) || 0) + 1);

			// Update inverted index
			if (!this.terms.has(term)) {
				this.terms.set(term, new Set());
			}
			this.terms.get(term)?.add(docId);
		}
		this.docTermFreq.set(docId, termFreq);

		// Update average document length
		const totalLength = Array.from(this.docTermFreq.values()).reduce(
			(sum, tf) => sum + Array.from(tf.values()).reduce((s, f) => s + f, 0),
			0,
		);
		this.avgDocLength = totalLength / this.documents.size;
	}

	/**
	 * Search the index using BM25 scoring
	 */
	search(
		query: string,
		limit: number,
	): Array<{ docId: string; score: number }> {
		const queryTerms = this.tokenize(query);
		const scores = new Map<string, number>();
		const N = this.documents.size;

		for (const term of queryTerms) {
			const matchingDocs = this.terms.get(term);
			if (!matchingDocs) continue;

			const df = matchingDocs.size;
			const idf = Math.log((N - df + 0.5) / (df + 0.5) + 1);

			for (const docId of matchingDocs) {
				const termFreq = this.docTermFreq.get(docId)?.get(term) || 0;
				const docLength = Array.from(
					this.docTermFreq.get(docId)?.values() || [],
				).reduce((s, f) => s + f, 0);

				const tf =
					(termFreq * (this.k1 + 1)) /
					(termFreq +
						this.k1 * (1 - this.b + this.b * (docLength / this.avgDocLength)));

				const score = idf * tf;
				scores.set(docId, (scores.get(docId) || 0) + score);
			}
		}

		return Array.from(scores.entries())
			.map(([docId, score]) => ({ docId, score }))
			.sort((a, b) => b.score - a.score)
			.slice(0, limit);
	}

	/**
	 * Get a document by ID
	 */
	getDocument(docId: string): GeneratedToolInfo | undefined {
		return this.documents.get(docId);
	}

	/**
	 * Get all documents
	 */
	getAllDocuments(): GeneratedToolInfo[] {
		return Array.from(this.documents.values());
	}

	/**
	 * Clear the index
	 */
	clear(): void {
		this.documents.clear();
		this.terms.clear();
		this.docTermFreq.clear();
		this.avgDocLength = 0;
	}
}

export class CodeSearchIndex {
	private bm25Index = new BM25Index();
	private documents: Map<string, GeneratedToolInfo> = new Map();

	/**
	 * Index a list of generated tools
	 */
	indexAll(tools: GeneratedToolInfo[]): void {
		this.clear();
		for (const tool of tools) {
			this.add(tool);
		}
	}

	/**
	 * Add a single tool to the index
	 */
	add(tool: GeneratedToolInfo): void {
		const docId = `${tool.serverId}.${tool.toolName}`;
		this.documents.set(docId, tool);
		this.bm25Index.add(tool);
	}

	/**
	 * Clear the index
	 */
	clear(): void {
		this.documents.clear();
		this.bm25Index.clear();
	}

	/**
	 * Search tools using the specified query
	 */
	search(query: SearchQuery): SearchResult {
		const mode = this.detectMode(query);

		if (mode === "regex") {
			return this.searchRegex(query.query, query.limit);
		} else {
			return this.searchBM25(query.query, query.limit);
		}
	}

	/**
	 * Detect search mode based on query
	 */
	private detectMode(query: SearchQuery): "bm25" | "regex" {
		if (query.mode === "bm25" || query.mode === "regex") {
			return query.mode;
		}

		// Auto-detect: if query contains regex special chars, use regex
		const regexChars = /[.*+?^${}()|[\]\\]/;
		if (regexChars.test(query.query)) {
			return "regex";
		}

		return "bm25";
	}

	/**
	 * Search using BM25 scoring
	 */
	private searchBM25(query: string, limit: number): SearchResult {
		const results = this.bm25Index.search(query, limit);

		const items: SearchResultItem[] = [];
		for (const { docId } of results) {
			const doc = this.documents.get(docId);
			if (!doc) continue;

			items.push({
				id: docId,
				serverId: doc.serverId,
				toolName: doc.toolName,
				functionName: doc.functionName,
				filePath: doc.filePath,
				description: doc.description,
				snippet: this.getSnippet(doc, query),
				returns: doc.returns,
			});
		}

		return {
			items,
			total: items.length,
		};
	}

	/**
	 * Search using regex pattern matching
	 */
	private searchRegex(pattern: string, limit: number): SearchResult {
		let regex: RegExp;
		try {
			regex = new RegExp(pattern, "gi");
		} catch {
			// Invalid regex, fall back to literal search
			regex = new RegExp(pattern.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "gi");
		}

		const items: SearchResultItem[] = [];

		for (const [docId, doc] of this.documents) {
			if (regex.test(doc.sourceCode) || regex.test(doc.description || "")) {
				items.push({
					id: docId,
					serverId: doc.serverId,
					toolName: doc.toolName,
					functionName: doc.functionName,
					filePath: doc.filePath,
					description: doc.description,
					snippet: this.getRegexSnippet(doc, regex),
					returns: doc.returns,
				});

				if (items.length >= limit) break;
			}

			// Reset regex state for next document
			regex.lastIndex = 0;
		}

		return {
			items,
			total: items.length,
		};
	}

	/**
	 * Get a compact parameter signature for the tool
	 */
	private getSnippet(doc: GeneratedToolInfo, _query: string): string {
		return this.extractCompactParams(doc);
	}

	/**
	 * Extract compact TypeScript-like parameter signature from Zod schema
	 * e.g., "{ url?: string, type?: 'a'|'b' }"
	 */
	private extractCompactParams(doc: GeneratedToolInfo): string {
		const source = doc.sourceCode;

		// Find the z.object({...}) schema
		const schemaMatch = source.match(
			/InputSchema\s*=\s*z\.object\(\{([\s\S]*?)\}\);/,
		);
		if (!schemaMatch) {
			return "{}";
		}

		const schemaContent = schemaMatch[1];
		const params: string[] = [];

		// Match each property: name: z.type().optional()
		const propRegex = /(\w+):\s*z\.(\w+)\(([^)]*)\)([^,\n]*)/g;
		let match: RegExpExecArray | null;

		while ((match = propRegex.exec(schemaContent)) !== null) {
			const [, name, zodType, enumValues, modifiers] = match;
			const isOptional = modifiers.includes(".optional()");

			let tsType: string;
			switch (zodType) {
				case "string":
					tsType = "string";
					break;
				case "number":
					tsType = "number";
					break;
				case "boolean":
					tsType = "boolean";
					break;
				case "enum": {
					// Extract enum values: ["a", "b"] -> "a"|"b"
					const values = enumValues.match(/\["([^"]+)"(?:,\s*"([^"]+)")*\]/);
					if (values) {
						const enumVals = enumValues.match(/"([^"]+)"/g);
						tsType = enumVals ? enumVals.map((v) => v).join("|") : "string";
					} else {
						tsType = "string";
					}
					break;
				}
				case "array":
					tsType = "array";
					break;
				case "object":
					tsType = "object";
					break;
				case "record":
					tsType = "Record<string, unknown>";
					break;
				default:
					tsType = "unknown";
			}

			params.push(`${name}${isOptional ? "?" : ""}: ${tsType}`);
		}

		if (params.length === 0) {
			return "{}";
		}

		return `{ ${params.join(", ")} }`;
	}

	/**
	 * Get a code snippet around regex matches (uses compact params)
	 */
	private getRegexSnippet(doc: GeneratedToolInfo, _regex: RegExp): string {
		return this.extractCompactParams(doc);
	}

	/**
	 * Get all indexed documents
	 */
	getAllTools(): GeneratedToolInfo[] {
		return Array.from(this.documents.values());
	}

	/**
	 * Get document count
	 */
	get size(): number {
		return this.documents.size;
	}
}

// Singleton instance
export const codeSearchIndex = new CodeSearchIndex();
