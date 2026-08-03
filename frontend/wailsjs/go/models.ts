export namespace app {
	
	export class ConvertResult {
	    Path: string;
	    Bytes: number;
	    Width: number;
	    Height: number;
	
	    static createFrom(source: any = {}) {
	        return new ConvertResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	        this.Bytes = source["Bytes"];
	        this.Width = source["Width"];
	        this.Height = source["Height"];
	    }
	}
	export class ImageInfo {
	    Path: string;
	    Width: number;
	    Height: number;
	
	    static createFrom(source: any = {}) {
	        return new ImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	        this.Width = source["Width"];
	        this.Height = source["Height"];
	    }
	}
	export class ImagesRequest {
	    Inputs: string[];
	    FrameMS: number;
	    Width: number;
	    Aspect: string;
	    Speed: number;
	    Reverse: boolean;
	    Boomerang: boolean;
	    Loop: string;
	    Colors: number;
	    Dither: string;
	    WebPQuality: number;
	    Format: string;
	    OutDir: string;
	    OutName: string;
	    OnExist: string;
	    TargetKB: number;
	
	    static createFrom(source: any = {}) {
	        return new ImagesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Inputs = source["Inputs"];
	        this.FrameMS = source["FrameMS"];
	        this.Width = source["Width"];
	        this.Aspect = source["Aspect"];
	        this.Speed = source["Speed"];
	        this.Reverse = source["Reverse"];
	        this.Boomerang = source["Boomerang"];
	        this.Loop = source["Loop"];
	        this.Colors = source["Colors"];
	        this.Dither = source["Dither"];
	        this.WebPQuality = source["WebPQuality"];
	        this.Format = source["Format"];
	        this.OutDir = source["OutDir"];
	        this.OutName = source["OutName"];
	        this.OnExist = source["OnExist"];
	        this.TargetKB = source["TargetKB"];
	    }
	}
	export class VideoInfo {
	    Path: string;
	    DurationMS: number;
	    Width: number;
	    Height: number;
	    FPS: number;
	
	    static createFrom(source: any = {}) {
	        return new VideoInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Path = source["Path"];
	        this.DurationMS = source["DurationMS"];
	        this.Width = source["Width"];
	        this.Height = source["Height"];
	        this.FPS = source["FPS"];
	    }
	}
	export class VideoRequest {
	    Input: string;
	    StartMS: number;
	    EndMS: number;
	    FPS: number;
	    Width: number;
	    SrcWidth: number;
	    SrcHeight: number;
	    Aspect: string;
	    Speed: number;
	    Reverse: boolean;
	    Boomerang: boolean;
	    Loop: string;
	    Colors: number;
	    Dither: string;
	    WebPQuality: number;
	    Format: string;
	    OutDir: string;
	    OutName: string;
	    OnExist: string;
	    TargetKB: number;
	
	    static createFrom(source: any = {}) {
	        return new VideoRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Input = source["Input"];
	        this.StartMS = source["StartMS"];
	        this.EndMS = source["EndMS"];
	        this.FPS = source["FPS"];
	        this.Width = source["Width"];
	        this.SrcWidth = source["SrcWidth"];
	        this.SrcHeight = source["SrcHeight"];
	        this.Aspect = source["Aspect"];
	        this.Speed = source["Speed"];
	        this.Reverse = source["Reverse"];
	        this.Boomerang = source["Boomerang"];
	        this.Loop = source["Loop"];
	        this.Colors = source["Colors"];
	        this.Dither = source["Dither"];
	        this.WebPQuality = source["WebPQuality"];
	        this.Format = source["Format"];
	        this.OutDir = source["OutDir"];
	        this.OutName = source["OutName"];
	        this.OnExist = source["OnExist"];
	        this.TargetKB = source["TargetKB"];
	    }
	}

}

