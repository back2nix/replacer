// source_complex.ts

export function simpleTsFunc(): void {
    console.log("New simpleTsFunc from source_complex.ts");
}

export const arrowTsFunc = (param1: string, param2: number): boolean => {
    console.log(`New arrowTsFunc from source_complex.ts: ${param1}, ${param2}`);
    return param1.length > param2;
};

class MyTsClass {
    constructor(private id: number) {}

    public classMethod(data: any): string {
        console.log(`New MyTsClass.classMethod from source_complex.ts, id: ${this.id}, data:`, data);
        return "new_class_method_result";
    }

    static staticTsMethod(): void {
        console.log("New MyTsClass.staticTsMethod from source_complex.ts");
    }
}

async function asyncTsFunc(name: string): Promise<string> {
    console.log(`New asyncTsFunc from source_complex.ts with ${name}`);
    await new Promise(resolve => setTimeout(resolve, 10));
    return `Async hello ${name} from source`;
}

// Not exported, should still be picked up if name matches
function utilityTsFunc() {
    console.log("New utilityTsFunc from source_complex.ts");
}

// This function is only in source_complex.ts, should be added to target
export function newSourceOnlyTsFunc(val: number): number {
    console.log("newSourceOnlyTsFunc from source_complex.ts, to be added", val);
    return val * val;
}

// export function commentedOutTsFunc(): void {
//   console.log("This should not be extracted or replaced");
// }

export function genericTsFunc<T>(arg: T): T {
    console.log("New genericTsFunc from source_complex.ts", arg);
    return arg;
}
