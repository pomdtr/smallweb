import { createStorage } from "unstorage"
import fsDriver from "unstorage/drivers/fs-lite"
import { createStorageHandler } from "unstorage/server";

const { SMALLWEB_DATA_DIR } = Deno.env.toObject();

if (!SMALLWEB_DATA_DIR) {
    console.error("SMALLWEB_DATA_DIR is not set.");
    Deno.exit(1);
}

const storage = createStorage({
    driver: fsDriver({
        base: SMALLWEB_DATA_DIR
    })
})

export default {
    fetch: createStorageHandler(storage),
}
