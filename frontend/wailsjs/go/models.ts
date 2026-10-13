export namespace main {
	
	export class CookieImportEntry {
	    id: string;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new CookieImportEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class CookieImportResult {
	    ok: boolean;
	    message?: string;
	    canceled?: boolean;
	    saved?: CookieImportEntry[];
	    dropped?: number;
	    dir?: string;
	
	    static createFrom(source: any = {}) {
	        return new CookieImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.canceled = source["canceled"];
	        this.saved = this.convertValues(source["saved"], CookieImportEntry);
	        this.dropped = source["dropped"];
	        this.dir = source["dir"];
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
	export class CookiePickResult {
	    ok: boolean;
	    message?: string;
	    canceled?: boolean;
	    path?: string;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new CookiePickResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.canceled = source["canceled"];
	        this.path = source["path"];
	        this.note = source["note"];
	    }
	}
	export class CookiePlatformStatus {
	    id: string;
	    name: string;
	    file: string;
	    imported: boolean;
	    count: number;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new CookiePlatformStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.file = source["file"];
	        this.imported = source["imported"];
	        this.count = source["count"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class CookieRemoveResult {
	    ok: boolean;
	    message?: string;
	    id?: string;
	    name?: string;
	
	    static createFrom(source: any = {}) {
	        return new CookieRemoveResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class CookieStatusResult {
	    dir: string;
	    platforms: CookiePlatformStatus[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new CookieStatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dir = source["dir"];
	        this.platforms = this.convertValues(source["platforms"], CookiePlatformStatus);
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
	export class EncoderStatus {
	    ok: boolean;
	    gpu: boolean;
	    encoder?: string;
	    label?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new EncoderStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.gpu = source["gpu"];
	        this.encoder = source["encoder"];
	        this.label = source["label"];
	        this.message = source["message"];
	    }
	}
	export class FFmpegInstallResult {
	    ok: boolean;
	    message?: string;
	    path?: string;
	    version?: string;
	    canceled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FFmpegInstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.canceled = source["canceled"];
	    }
	}
	export class FilePickResult {
	    ok: boolean;
	    message?: string;
	    path?: string;
	    version?: string;
	    canceled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FilePickResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.path = source["path"];
	        this.version = source["version"];
	        this.canceled = source["canceled"];
	    }
	}
	export class ProxyTestResult {
	    ok: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}
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
	export class SystemStatus {
	    engineReady: boolean;
	    enginePath: string;
	    engineError: string;
	    ffmpegReady: boolean;
	    ffmpegPath: string;
	    ffmpegError: string;
	    ffmpegVersion: string;
	    proxyChecked: boolean;
	    proxyReady: boolean;
	    proxyMessage: string;
	    dataDir: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engineReady = source["engineReady"];
	        this.enginePath = source["enginePath"];
	        this.engineError = source["engineError"];
	        this.ffmpegReady = source["ffmpegReady"];
	        this.ffmpegPath = source["ffmpegPath"];
	        this.ffmpegError = source["ffmpegError"];
	        this.ffmpegVersion = source["ffmpegVersion"];
	        this.proxyChecked = source["proxyChecked"];
	        this.proxyReady = source["proxyReady"];
	        this.proxyMessage = source["proxyMessage"];
	        this.dataDir = source["dataDir"];
	    }
	}
	
	export class UpdateDownloadResult {
	    ok: boolean;
	    error?: string;
	    path?: string;
	    size?: number;
	    reused?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateDownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.reused = source["reused"];
	    }
	}
	export class UpdateInstallResult {
	    ok: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.error = source["error"];
	    }
	}
	export class UpdateResult {
	    ok: boolean;
	    hasUpdate: boolean;
	    current: string;
	    latest: string;
	    notes: string;
	    url: string;
	    pageUrl: string;
	    source: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.hasUpdate = source["hasUpdate"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	        this.pageUrl = source["pageUrl"];
	        this.source = source["source"];
	        this.error = source["error"];
	    }
	}

}

