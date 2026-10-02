package alluvial

import (
	"strconv"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
)

const mergedConstantPathPrefix = "\x00constant:"

func applyConstantBands(
	columns []Column,
	mode config.ConstantBandsMode,
) ([]Column, map[string]struct{}) {
	if mode == "" {
		return columns, nil
	}

	constant := constantPaths(columns)

	switch mode {
	case config.ConstantBandsHide:
		return hideConstantPaths(columns, constant), nil
	case config.ConstantBandsMute:
		return columns, constant
	case config.ConstantBandsMerge:
		return mergeConstantPaths(columns, constant)
	default:
		return columns, nil
	}
}

func constantPaths(columns []Column) map[string]struct{} {
	if len(columns) == 0 {
		return nil
	}

	first := valuesByPath(columns[0].Values)
	constant := make(map[string]struct{}, len(first))

	for directoryPath, width := range first {
		isConstant := true

		for _, column := range columns[1:] {
			value, exists := valuesByPath(column.Values)[directoryPath]
			if !exists || value != width {
				isConstant = false

				break
			}
		}

		if isConstant {
			constant[directoryPath] = struct{}{}
		}
	}

	return constant
}

func hideConstantPaths(columns []Column, constant map[string]struct{}) []Column {
	result := make([]Column, len(columns))

	for columnIndex, column := range columns {
		values := make([]Value, 0, len(column.Values))

		for _, value := range column.Values {
			if _, isConstant := constant[value.Path]; !isConstant {
				values = append(values, value)
			}
		}

		result[columnIndex] = Column{Reference: column.Reference, Values: values}
	}

	return result
}

type constantBandGroup struct {
	path    string
	members []string
	muted   bool
}

func mergeConstantPaths(
	columns []Column,
	constant map[string]struct{},
) ([]Column, map[string]struct{}) {
	groups := constantBandGroups(columns, constant)
	muted := make(map[string]struct{})
	result := make([]Column, len(columns))

	for columnIndex, column := range columns {
		widths := valuesByPath(column.Values)
		values := make([]Value, 0, len(groups))

		for _, group := range groups {
			width := 0.0
			present := false

			for _, member := range group.members {
				value, exists := widths[member]
				if exists {
					present = true
					width += value
				}
			}

			if !present {
				continue
			}

			values = append(values, Value{Path: group.path, Width: width})
			if group.muted {
				muted[group.path] = struct{}{}
			}
		}

		result[columnIndex] = Column{Reference: column.Reference, Values: values}
	}

	return result, muted
}

func constantBandGroups(columns []Column, constant map[string]struct{}) []constantBandGroup {
	allPaths := make(map[string]float64)
	for _, column := range columns {
		for _, value := range column.Values {
			allPaths[value.Path] = value.Width
		}
	}

	paths := unionPaths(allPaths, nil)
	groups := make([]constantBandGroup, 0, len(paths))
	constantGroup := 0

	for index := 0; index < len(paths); {
		directoryPath := paths[index]
		if _, isConstant := constant[directoryPath]; !isConstant {
			groups = append(groups, constantBandGroup{
				path:    directoryPath,
				members: []string{directoryPath},
			})
			index++

			continue
		}

		end := index + 1
		for end < len(paths) {
			if _, isConstant := constant[paths[end]]; !isConstant {
				break
			}

			end++
		}

		groups = append(groups, constantBandGroup{
			path:    mergedConstantPathPrefix + strconv.Itoa(constantGroup),
			members: paths[index:end],
			muted:   true,
		})
		constantGroup++
		index = end
	}

	return groups
}
