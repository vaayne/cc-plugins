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
      this.terms.get(term)!.add(docId);
    }
    this.docTermFreq.set(docId, termFreq);

    // Update average document length
    const totalLength = Array.from(this.docTermFreq.values()).reduce(
      (sum, tf) => sum + Array.from(tf.values()).reduce((s, f) => s + f, 0),
      0
    );
    this.avgDocLength = totalLength / this.documents.size;
  }

  /**
   * Search the index using BM25 scoring
   */
  search(query: string, limit: number): Array<{ docId: string; score: number }> {
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
        const docLength = Array.from(this.docTermFreq.get(docId)?.values() || []).reduce(
          (s, f) => s + f,
          0
        );

        const tf =
          (termFreq * (this.k1 + 1)) /
          (termFreq + this.k1 * (1 - this.b + this.b * (docLength / this.avgDocLength)));

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
   * Get a useful code snippet that includes function signature and input schema
   */
  private getSnippet(doc: GeneratedToolInfo, _query: string): string {
    return this.extractUsefulSnippet(doc);
  }

  /**
   * Extract the most useful parts of the tool: signature and input schema
   */
  private extractUsefulSnippet(doc: GeneratedToolInfo): string {
    const lines = doc.sourceCode.split("\n");
    const snippetParts: string[] = [];

    // Find and extract the input schema (z.object({...}))
    let inSchema = false;
    let braceCount = 0;
    let schemaLines: string[] = [];

    for (const line of lines) {
      // Start capturing input schema
      if (line.includes("InputSchema = z.object(")) {
        inSchema = true;
        braceCount = 0;
      }

      if (inSchema) {
        schemaLines.push(line);
        braceCount += (line.match(/\{/g) || []).length;
        braceCount -= (line.match(/\}/g) || []).length;

        // End of schema object
        if (braceCount <= 0 && schemaLines.length > 0) {
          inSchema = false;
          break;
        }
      }
    }

    // Add input schema if found
    if (schemaLines.length > 0) {
      snippetParts.push("// Input Schema:");
      snippetParts.push(schemaLines.join("\n"));
    }

    // Find and extract function signature with its JSDoc
    const funcMatch = doc.sourceCode.match(
      /\/\*\*[\s\S]*?\*\/\s*export async function \w+\([^)]*\):\s*Promise<[^>]+>/
    );
    if (funcMatch) {
      snippetParts.push("\n// Function:");
      snippetParts.push(funcMatch[0]);
    } else {
      // Try simpler function signature match
      const simpleMatch = doc.sourceCode.match(/export async function \w+\([^)]*\):\s*Promise<[^>]+>/);
      if (simpleMatch) {
        snippetParts.push("\n// Function:");
        snippetParts.push(simpleMatch[0]);
      }
    }

    if (snippetParts.length === 0) {
      // Fallback: first 300 chars
      return doc.sourceCode.slice(0, 300) + (doc.sourceCode.length > 300 ? "..." : "");
    }

    return snippetParts.join("\n");
  }

  /**
   * Get a code snippet around regex matches (now uses same useful snippet)
   */
  private getRegexSnippet(doc: GeneratedToolInfo, _regex: RegExp): string {
    return this.extractUsefulSnippet(doc);
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
