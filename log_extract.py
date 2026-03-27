import argparse

def main():
    parser = argparse.ArgumentParser(description="Filter log lines by substring(s).")
    parser.add_argument("--input", required=True, help="input log file path")
    parser.add_argument("--output", required=True, help="output file path")
    parser.add_argument("--pattern", action="append", required=True,
                        help="substring to match (can be provided multiple times)")
    parser.add_argument("--mode", choices=["any", "all"], default="all",
                        help="any: Output the line if it contains at least one of the patterns;all: Output the line if it contains all the patterns")
    parser.add_argument("--encoding", default="utf-8", help="file encoding (default: utf-8)")
    args = parser.parse_args()

    patterns = args.pattern

    with open(args.input, "r", encoding=args.encoding, errors="replace") as fin, \
         open(args.output, "w", encoding=args.encoding) as fout:
        for line in fin:
            line_has = [(p in line) for p in patterns]
            if args.mode == "any":
                if any(line_has):
                    fout.write(line)
            else:  # mode == "all"
                if all(line_has):
                    fout.write(line)

if __name__ == "__main__":
    main()

