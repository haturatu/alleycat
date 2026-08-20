import { describe, expect, test } from "vitest";

import { __test__ } from "./mediaUpload";

describe("hashFileSHA256", () => {
  test("returns the SHA-256 digest for a file", async () => {
    const file = new File(["hello"], "hello.txt");

    await expect(__test__.hashFileSHA256(file)).resolves.toBe(
      "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
    );
  });
});

describe("toRelativeMediaURL", () => {
  test("converts pocketbase file URLs to relative paths", () => {
    expect(
      __test__.toRelativeMediaURL(
        "http://127.0.0.1:8090/api/files/pbc_2708086759/xywk7dopjdypynj/example.jpg"
      )
    ).toBe("/api/files/pbc_2708086759/xywk7dopjdypynj/example.jpg");
  });

  test("keeps upload paths relative", () => {
    expect(__test__.toRelativeMediaURL("https://example.com/uploads/example.jpg")).toBe(
      "/uploads/example.jpg"
    );
  });

  test("leaves non-media absolute URLs unchanged", () => {
    expect(__test__.toRelativeMediaURL("https://git.soulminingrig.com/")).toBe(
      "https://git.soulminingrig.com/"
    );
  });
});
