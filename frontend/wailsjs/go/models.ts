export namespace main {
	
	export class Settings {
	    quality: string;
	    downloadType: string;
	    downloadDir: string;
	    useProxy: boolean;
	    proxyURL: string;
	    autoDetectProxy: boolean;
	    useBrowserCookie: boolean;
	    cookieSource: string;
	    cookieBrowser: string;
	    cookieFile: string;
	    ffmpegPath: string;
	    highFrameRate: boolean;
	    openOnComplete: boolean;
	    maxConcurrent: number;
	    autoRetry: number;
	    videoCodec: string;
	    hevcEncoder: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.quality = source["quality"];
	        this.downloadType = source["downloadType"];
	        this.downloadDir = source["downloadDir"];
	        this.useProxy = source["useProxy"];
	        this.proxyURL = source["proxyURL"];
	        this.autoDetectProxy = source["autoDetectProxy"];
	        this.useBrowserCookie = source["useBrowserCookie"];
	        this.cookieSource = source["cookieSource"];
	        this.cookieBrowser = source["cookieBrowser"];
	        this.cookieFile = source["cookieFile"];
	        this.ffmpegPath = source["ffmpegPath"];
	        this.highFrameRate = source["highFrameRate"];
	        this.openOnComplete = source["openOnComplete"];
	        this.maxConcurrent = source["maxConcurrent"];
	        this.autoRetry = source["autoRetry"];
	        this.videoCodec = source["videoCodec"];
	        this.hevcEncoder = source["hevcEncoder"];
	    }
	}
	export class Task {
	    id: number;
	    url: string;
	    title: string;
	    status: string;
	    phase?: string;
	    progress: number;
	    quality: string;
	    // Go type: time
	    createdAt: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.phase = source["phase"];
	        this.progress = source["progress"];
	        this.quality = source["quality"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.error = source["error"];
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
	    tasks: Task[];
	    settings: Settings;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tasks = this.convertValues(source["tasks"], Task);
	        this.settings = this.convertValues(source["settings"], Settings);
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

