declare namespace Smallweb {
    interface App {
        fetch?(req: Request): Response | Promise<Response>;
        run?: (args: string[]) => void | Promise<void>;
    }
}
