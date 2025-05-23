// target_complex.ts

export function simpleTsFunc(): void {
    console.log("Old simpleTsFunc from target_complex.ts");
}

export const arrowTsFunc = (param1: string, param2: number): boolean => {
    console.log(`Old arrowTsFunc from target_complex.ts: ${param1}, ${param2}`);
    return false;
};

class MyTsClass {
    constructor(private id: number) {}

    public classMethod(data: any): string {
        console.log(`Old MyTsClass.classMethod from target_complex.ts, id: ${this.id}, data:`, data);
        return "old_class_method_result";
    }

    static staticTsMethod(): void {
        console.log("Old MyTsClass.staticTsMethod from target_complex.ts");
    }

    public keepThisMethod(): void {
        console.log("MyTsClass.keepThisMethod from target_complex.ts - I should remain.");
    }
}

async function asyncTsFunc(name: string): Promise<string> {
    console.log(`Old asyncTsFunc from target_complex.ts with ${name}`);
    await new Promise(resolve => setTimeout(resolve, 5));
    return `Async hello ${name} from target`;
}

function utilityTsFunc() {
    console.log("Old utilityTsFunc from target_complex.ts");
}

export function targetSpecificTsFunc(): void {
    console.log("targetSpecificTsFunc in target_complex.ts");
}

// A commented out version, should not be touched
// export function simpleTsFunc(): void {
//    console.log("Commented out simpleTsFunc");
// }

export function genericTsFunc<T>(arg: T): T {
    console.log("Old genericTsFunc from target_complex.ts", arg);
    return arg;
}

// Some code after all functions
export const endMarkerTargetComplexTs = true;
