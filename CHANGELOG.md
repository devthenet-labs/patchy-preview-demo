# Changelog

Every pull request adds a line under `## Unreleased` that names its own number as `#<number>`; the `changelog` check
fails without one.

## Unreleased

- Add this changelog and the `changelog` pull-request check (#8).
- Add a static "Deployed by patchy" footer to the welcome card (#9).
- Show the build time in the card footer (#10).
- Add a subtitle under the welcome heading (#11).
- Change the card footer to "preview-demo <version>" (#12).
- Style the footer in muted gray, add an `aria-label`, and mention it in the README (#17).
- Show the current year in the card footer after the version (#18).
- Sign the published agent image with the KMS key (#19).
- Add "signed" to the card footer after the year (#20).
- Add a `GET /id` endpoint returning a random UUID (#21).
- Title-case the welcome greeting with golang.org/x/text/cases (#22).
- Add a GET /semver endpoint validating versions with golang.org/x/mod/semver (#23).
- Add a GET /slug endpoint using github.com/gosimple/slug (#24).
- Add a GET /wordcount endpoint counting words and Unicode characters (#25).
