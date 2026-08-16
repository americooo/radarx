export namespace model {
	
	export class Asset {
	    kind: string;
	    key: string;
	    host?: string;
	    ip?: string[];
	    port?: number;
	    scheme?: string;
	    status_code?: number;
	    title?: string;
	    server?: string;
	    cert_cn?: string;
	    cert_expiry?: string;
	
	    static createFrom(source: any = {}) {
	        return new Asset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.key = source["key"];
	        this.host = source["host"];
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.scheme = source["scheme"];
	        this.status_code = source["status_code"];
	        this.title = source["title"];
	        this.server = source["server"];
	        this.cert_cn = source["cert_cn"];
	        this.cert_expiry = source["cert_expiry"];
	    }
	}
	export class Change {
	    type: string;
	    kind: string;
	    key: string;
	    before?: Asset;
	    after?: Asset;
	    fields?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.kind = source["kind"];
	        this.key = source["key"];
	        this.before = this.convertValues(source["before"], Asset);
	        this.after = this.convertValues(source["after"], Asset);
	        this.fields = source["fields"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DiffResult {
	    target_id: string;
	    root: string;
	    // Go type: time
	    at: any;
	    changes: Change[];
	
	    static createFrom(source: any = {}) {
	        return new DiffResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_id = source["target_id"];
	        this.root = source["root"];
	        this.at = this.convertValues(source["at"], null);
	        this.changes = this.convertValues(source["changes"], Change);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Snapshot {
	    target_id: string;
	    root: string;
	    // Go type: time
	    taken_at: any;
	    assets: Asset[];
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_id = source["target_id"];
	        this.root = source["root"];
	        this.taken_at = this.convertValues(source["taken_at"], null);
	        this.assets = this.convertValues(source["assets"], Asset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Target {
	    id: string;
	    root: string;
	    label: string;
	    enabled: boolean;
	    interval_m: number;
	    wordlist: string[];
	    // Go type: time
	    added_at: any;
	    // Go type: time
	    last_scan_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.root = source["root"];
	        this.label = source["label"];
	        this.enabled = source["enabled"];
	        this.interval_m = source["interval_m"];
	        this.wordlist = source["wordlist"];
	        this.added_at = this.convertValues(source["added_at"], null);
	        this.last_scan_at = this.convertValues(source["last_scan_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

