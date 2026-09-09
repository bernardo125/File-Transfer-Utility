export namespace ui {
	
	export class FileInfoDTO {
	    path: string;
	    name: string;
	    size: number;
	    hash: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfoDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.hash = source["hash"];
	    }
	}

}

