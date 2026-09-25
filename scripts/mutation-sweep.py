#!/usr/bin/env python3
"""Mutation sweep: break each behavior and confirm the targeting test fails.

A test that still passes under mutation is a false pass.
"""
import subprocess
import sys

MUTATIONS = [
    # (criterion, file, old, new, tests, extra go test args)
    ("V1 config-time rejection", "cuid2.go",
     "math.IsNaN(r) || r < 0 || r >= 1",
     "math.IsNaN(r) || r < 0",
     ["TestWithRandomFuncRejectsOutOfRangeValues"], []),

    ("V2 lifetime wrapper", "cuid2.go",
     'panic("Error: the provided random function returned a value outside the range [0, 1)")',
     "_ = v",
     ["TestCustomRandomFuncOutOfContractPanics"], []),

    ("V3 nil counter", "cuid2.go",
     "if sessionCounter == nil {",
     "if false {",
     ["TestWithSessionCounterRejectsNil"], []),

    ("V4 nil generator", "cuid2.go",
     "return nil, applyErr",
     'return func() string { return "" }, applyErr',
     ["TestInitReturnsNilGeneratorOnError"], []),

    ("V6 fingerprint truncation", "cuid2.go",
     "return hash(sourceString)[:MaxIdLength]",
     "return hash(sourceString)[1:]",
     ["TestFingerprintLength", "TestCreatingFingerprintWithEnvKeyString"], []),

    ("V8 precompiled regex", "cuid2.go",
     "return cuidRegex.MatchString(cuid) && length >= MinIdLength && length <= MaxIdLength",
     'm, _ := regexp.MatchString("^[a-z][0-9a-z]+$", cuid)\n\treturn m && length >= MinIdLength && length <= MaxIdLength',
     ["TestIsCuidDoesNotCompilePerCall"], []),

    ("V9 env guard", "cuid2.go",
     "if idx := strings.IndexByte(variable, '='); idx >= 0 {",
     "if idx := strings.IndexByte(variable, '='); true {",
     ["TestEnvironmentKeyStringSkipsEntriesWithoutSeparator"], []),

    ("V10 histogram clamp", "collision_test.go",
     "if index >= HistogramBuckets {",
     "if false {",
     ["TestHistogramBucketClampsTopOfRange"], ["-tags=integration"]),
]


def main():
    results = []
    for name, path, old, new, tests, extra in MUTATIONS:
        original = open(path).read()
        if old not in original:
            print(f"SKIP  {name}: pattern not found in {path}")
            results.append((name, None))
            continue
        try:
            open(path, "w").write(original.replace(old, new, 1))
            cmd = ["go", "test"] + extra + ["-run", "|".join(tests), "./..."]
            run = subprocess.run(cmd, capture_output=True, text=True)
            collided = run.returncode != 0
            verdict = "COLLIDED" if collided else "FALSE PASS"
            print(f"{verdict:10} {name}")
            if not collided:
                print(run.stdout[-800:])
            results.append((name, collided))
        finally:
            open(path, "w").write(original)

    bad = [n for n, c in results if c is False]
    skipped = [n for n, c in results if c is None]
    print()
    print(f"mutations: {len(results)}  collided: {sum(1 for _, c in results if c)}  "
          f"false-pass: {len(bad)}  skipped: {len(skipped)}")
    if bad:
        print("FALSE PASSES:", bad)
        sys.exit(1)


if __name__ == "__main__":
    main()
