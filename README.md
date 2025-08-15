# tfpdiff
Terraform plan differ. Useful for helm releases values. `terraform plan` looks at whole values as a single string. This app parses it as JSON and shows a diff for a specific field, instead of the whole string.

## Installation
`go install github.com/svennjegac/tfpdiff/cmd/tfpdiff`

## Usage
- Specify file as a flag: `tfpdiff -f plan.json`
- Read terraform json plan from stdin: `tfpdiff`
- Optionally, turn on the color output: `tfpdiff -f plan.json -c` or `tfpdiff -c`
