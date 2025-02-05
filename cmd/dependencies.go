/*
Copyright © 2021 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/format"
	"github.com/spf13/cobra"
)

var dependencies_stackName *string

// dependenciesCmd represents the dependencies command
var dependenciesCmd = &cobra.Command{
	Use:   "dependencies",
	Short: "Show dependencies between your stacks",
	Long: `This will show your stacks and any dependencies that exist between them.

Dependencies can prevent updates from happening or prevent a stack from getting deleted.
Right now dependencies being shown are export values that are imperted by other stacks. Upcoming is support for showing nested stacks.

This function supports the "dot" output format, which outputs the dependencies in a form you can turn into a graphical environment using a tool like graphviz.

$ fog dependencies --output dot --stackname "*dev*" | dot -Tpng -o cfn-deps.png

`,
	Run: showDependencies,
}

func init() {
	rootCmd.AddCommand(dependenciesCmd)
	dependencies_stackName = dependenciesCmd.Flags().StringP("stackname", "n", "", "Name, ID, or wildcard filter for the stack (optional)")
}

func showDependencies(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	emptystring := ""
	stacks, err := lib.GetCfnStacks(&emptystring, awsConfig.CloudformationClient())
	if err != nil {
		panic(err)
	}
	keys := []string{"Stack", "Description", "Imported By"}
	subtitle := "All stacks"
	if *dependencies_stackName != "" {
		subtitle = fmt.Sprintf("Stacks filtered by for %v", *dependencies_stackName)
	}
	title := fmt.Sprintf("%v in account %v for region %v", subtitle, awsConfig.AccountID, awsConfig.Region)
	output := format.OutputArray{Keys: keys, Title: title}
	output.SortKey = "Stack"
	if settings.GetLCString("output") == "dot" {
		dotcolumns := config.DotColumns{
			From: "Stack",
			To:   "Imported By",
		}
		settings.DotColumns = &dotcolumns
	}
	stackfilter := []string{}
	if *dependencies_stackName != "" {
		stackfilter = unique(getFilteredStacks(*dependencies_stackName, &stacks))
	}
	for stackname, stack := range stacks {
		if *dependencies_stackName != "" && !stringInSlice(stackname, stackfilter) {
			continue
		}
		content := make(map[string]string)
		content["Stack"] = stack.Name
		content["Description"] = stack.Description
		content["Imported By"] = strings.Join(unique(stack.ImportedBy), settings.GetSeparator())
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)
}

func getFilteredStacks(stackfilter string, stacks *map[string]lib.CfnStack) []string {
	result := []string{}
	stackRegex := "^" + strings.Replace(stackfilter, "*", ".*", -1) + "$"
	for stackname, stack := range *stacks {
		if strings.Contains(stackfilter, "*") {
			if matched, err := regexp.MatchString(stackRegex, stackname); matched && err == nil {
				result = append(result, stackname)
				for _, importedstack := range stack.ImportedBy {
					result = append(result, getFilteredStacks(importedstack, stacks)...)
				}
			} else {
				for _, importstack := range stack.ImportedBy {
					if matched, err := regexp.MatchString(stackRegex, importstack); matched && err == nil {
						result = append(result, stackname)
					}
				}
			}
		} else {
			if stackname == stackfilter {
				result = append(result, stackname)
				for _, importedstack := range stack.ImportedBy {
					result = append(result, getFilteredStacks(importedstack, stacks)...)
				}
			}
			if stringInSlice(stackfilter, stack.ImportedBy) {
				result = append(result, stackname)
			}
		}
	}
	return result
}
