import { describe, expect, test } from "vitest";
import { formatDateTimeLocalInput, localInputToISOString, parseServerDateTime } from "./datetime";

describe("datetime helpers", () => {
  const formatLocal = (date: Date) =>
    [
      `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`,
      `${String(date.getHours()).padStart(2, "0")}:${String(date.getMinutes()).padStart(2, "0")}`,
    ].join("T");

  test("parseServerDateTime supports PocketBase space-separated UTC timestamps", () => {
    const parsed = parseServerDateTime("2026-04-19 21:11:00.000Z");
    expect(parsed?.toISOString()).toBe("2026-04-19T21:11:00.000Z");
  });

  test("formatDateTimeLocalInput renders local datetime-local string", () => {
    const date = new Date("2026-04-19T21:30:00.000Z");
    expect(formatDateTimeLocalInput("2026-04-19T21:30:00.000Z")).toBe(formatLocal(date));
  });

  test("localInputToISOString converts local datetime-local input back to UTC", () => {
    const realDate = Date;
    class MockDate extends Date {
      constructor(...args: ConstructorParameters<typeof Date>) {
        if (args.length === 0) {
          super();
          return;
        }
        if (args.length >= 2 && typeof args[0] === "number") {
          const [year, month, day = 1, hour = 0, minute = 0, second = 0, ms = 0] = args as number[];
          super(Date.UTC(year, month, day, hour-9, minute, second, ms));
          return;
        }
        super(args[0] as string | number | Date);
      }
    }
    // @ts-expect-error test override
    globalThis.Date = MockDate;
    try {
      expect(localInputToISOString("2026-04-20T06:30")).toBe("2026-04-19T21:30:00.000Z");
    } finally {
      globalThis.Date = realDate;
    }
  });
});
