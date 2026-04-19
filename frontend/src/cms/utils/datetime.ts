const serverDateLayouts = [
  /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2})(\.\d{1,3})?)?Z$/,
  /^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2})(?::(\d{2})(\.\d{1,3})?)?Z$/,
];

const pad = (value: number) => String(value).padStart(2, "0");

export const parseServerDateTime = (value?: string) => {
  const input = value?.trim() || "";
  if (!input) return null;

  for (const pattern of serverDateLayouts) {
    const match = input.match(pattern);
    if (!match) continue;
    const [, year, month, day, hour, minute, second = "0"] = match;
    return new Date(Date.UTC(
      Number(year),
      Number(month) - 1,
      Number(day),
      Number(hour),
      Number(minute),
      Number(second),
    ));
  }

  const parsed = new Date(input);
  if (Number.isNaN(parsed.getTime())) {
    return null;
  }
  return parsed;
};

export const formatDateTimeLocalInput = (value?: string) => {
  const parsed = parseServerDateTime(value);
  if (!parsed) return "";

  return [
    `${parsed.getFullYear()}-${pad(parsed.getMonth() + 1)}-${pad(parsed.getDate())}`,
    `${pad(parsed.getHours())}:${pad(parsed.getMinutes())}`,
  ].join("T");
};

export const localInputToISOString = (value?: string) => {
  const input = value?.trim() || "";
  if (!input) return "";

  const match = input.match(/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/);
  if (!match) return "";

  const [, year, month, day, hour, minute] = match;
  return new Date(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
    0,
    0,
  ).toISOString();
};
