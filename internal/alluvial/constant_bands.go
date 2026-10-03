package alluvial

import "github.com/theunrepentantgeek/code-visualizer/internal/config"

const mergedConstantPathSuffix = "\x00constant"

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
	widthsByColumn := make([]map[string]float64, len(columns)-1)

	for index, column := range columns[1:] {
		widthsByColumn[index] = valuesByPath(column.Values)
	}

	for directoryPath, width := range first {
		if hasConstantWidth(directoryPath, width, widthsByColumn) {
			constant[directoryPath] = struct{}{}
		}
	}

	return constant
}

func hasConstantWidth(
	directoryPath string,
	width float64,
	widthsByColumn []map[string]float64,
) bool {
	for _, widths := range widthsByColumn {
		value, exists := widths[directoryPath]
		if !exists || value != width {
			return false
		}
	}

	return true
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
	result := make([]Column, len(columns))

	for columnIndex, column := range columns {
		result[columnIndex] = Column{
			Reference: column.Reference,
			Values:    mergedColumnValues(column.Values, groups),
		}
	}

	return result, mutedGroupPaths(groups)
}

func mergedColumnValues(values []Value, groups []constantBandGroup) []Value {
	widths := valuesByPath(values)

	originalValues := make(map[string]Value, len(values))
	for _, value := range values {
		originalValues[value.Path] = value
	}

	result := make([]Value, 0, len(groups))

	for _, group := range groups {
		if !group.muted && len(group.members) == 1 {
			if value, present := originalValues[group.members[0]]; present {
				result = append(result, value)
			}

			continue
		}

		width, present := groupWidth(group, widths)
		if present {
			result = append(result, Value{Path: group.path, Width: width})
		}
	}

	return result
}

func groupWidth(group constantBandGroup, widths map[string]float64) (float64, bool) {
	width := 0.0
	present := false

	for _, member := range group.members {
		value, exists := widths[member]
		if exists {
			present = true
			width += value
		}
	}

	return width, present
}

func mutedGroupPaths(groups []constantBandGroup) map[string]struct{} {
	muted := make(map[string]struct{})

	for _, group := range groups {
		if group.muted {
			muted[group.path] = struct{}{}
		}
	}

	return muted
}

func constantBandGroups(columns []Column, constant map[string]struct{}) []constantBandGroup {
	paths := selectedPaths(columns)
	groups := make([]constantBandGroup, 0, len(paths))

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

		end := constantRunEnd(paths, index+1, constant)
		groups = append(groups, constantBandGroup{
			path:    paths[index] + mergedConstantPathSuffix,
			members: paths[index:end],
			muted:   true,
		})
		index = end
	}

	return groups
}

func selectedPaths(columns []Column) []string {
	allPaths := make(map[string]float64)

	for _, column := range columns {
		for _, value := range column.Values {
			allPaths[value.Path] = value.Width
		}
	}

	return unionPaths(allPaths, nil)
}

func constantRunEnd(paths []string, start int, constant map[string]struct{}) int {
	end := start

	for end < len(paths) {
		if _, isConstant := constant[paths[end]]; !isConstant {
			break
		}

		end++
	}

	return end
}
