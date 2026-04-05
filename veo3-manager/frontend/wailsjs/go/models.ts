export namespace chrome {
	
	export class BrowserInfo {
	    status: string;
	    chromePath: string;
	    profilePath: string;
	    debugPort: number;
	    webSocketURL: string;
	    version: string;
	    stealth: boolean;
	    stealthMods: string[];
	
	    static createFrom(source: any = {}) {
	        return new BrowserInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.chromePath = source["chromePath"];
	        this.profilePath = source["profilePath"];
	        this.debugPort = source["debugPort"];
	        this.webSocketURL = source["webSocketURL"];
	        this.version = source["version"];
	        this.stealth = source["stealth"];
	        this.stealthMods = source["stealthMods"];
	    }
	}

}

export namespace config {
	
	export class AppConfig {
	    chromePath: string;
	    userDataDir: string;
	    downloadDir: string;
	    debugPort: string;
	    aspectRatio: string;
	    model: string;
	    outputCount: number;
	    dbPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chromePath = source["chromePath"];
	        this.userDataDir = source["userDataDir"];
	        this.downloadDir = source["downloadDir"];
	        this.debugPort = source["debugPort"];
	        this.aspectRatio = source["aspectRatio"];
	        this.model = source["model"];
	        this.outputCount = source["outputCount"];
	        this.dbPath = source["dbPath"];
	    }
	}

}

export namespace database {
	
	export class Task {
	    id: string;
	    prompt: string;
	    status: string;
	    aspectRatio: string;
	    model: string;
	    outputCount: number;
	    mediaIds: string[];
	    videoPaths: string[];
	    errorMessage: string;
	    seed: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    completedAt: sql.NullTime;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.prompt = source["prompt"];
	        this.status = source["status"];
	        this.aspectRatio = source["aspectRatio"];
	        this.model = source["model"];
	        this.outputCount = source["outputCount"];
	        this.mediaIds = source["mediaIds"];
	        this.videoPaths = source["videoPaths"];
	        this.errorMessage = source["errorMessage"];
	        this.seed = source["seed"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], sql.NullTime);
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
	export class TaskFilter {
	    status: string;
	    search: string;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.search = source["search"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	export class TaskStats {
	    total: number;
	    pending: number;
	    processing: number;
	    completed: number;
	    failed: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.pending = source["pending"];
	        this.processing = source["processing"];
	        this.completed = source["completed"];
	        this.failed = source["failed"];
	    }
	}

}

export namespace sql {
	
	export class NullTime {
	    // Go type: time
	    Time: any;
	    Valid: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NullTime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Time = this.convertValues(source["Time"], null);
	        this.Valid = source["Valid"];
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

