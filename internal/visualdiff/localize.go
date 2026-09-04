package visualdiff

import (
	"fmt"
	"image"
	"math"
	"sort"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

// LocalizationOptions controls clustering of changed pixels and DOM element correlation.
type LocalizationOptions struct {
	ChannelTolerance uint8
	ClusterPadding   int     // pixels to merge neighboring changed areas (default: 8)
	MinClusterPixels int     // ignore tiny noise spots (default: 4)
	MinOverlapRatio  float64 // minimum intersection area with element (default: 0.1)
}

// DefaultLocalizationOptions returns production default clustering settings.
func DefaultLocalizationOptions() LocalizationOptions {
	return LocalizationOptions{
		ChannelTolerance: 0,
		ClusterPadding:   12,
		MinClusterPixels: 4,
		MinOverlapRatio:  0.05,
	}
}

// LocalizeDifferences performs in-memory comparison, clusters disjoint changed regions,
// and intersects them with DOM elements to generate localized visual findings.
func LocalizeDifferences(a, b *image.RGBA, elements []evidence.ElementRef, opts LocalizationOptions) ([]evidence.VisualRegion, []evidence.VisualFinding, error) {
	if a == nil || b == nil {
		return nil, nil, fmt.Errorf("visualdiff: nil RGBA input")
	}
	if a.Bounds().Dx() != b.Bounds().Dx() || a.Bounds().Dy() != b.Bounds().Dy() {
		return nil, nil, fmt.Errorf("visualdiff: dimensions differ: %v vs %v", a.Bounds(), b.Bounds())
	}

	diffRes, err := CompareRGBA(a, b, Options{ChannelTolerance: opts.ChannelTolerance})
	if err != nil {
		return nil, nil, err
	}
	if diffRes.ChangedPixels == 0 {
		return nil, nil, nil
	}

	if opts.ClusterPadding <= 0 {
		opts.ClusterPadding = 8
	}
	if opts.MinClusterPixels <= 0 {
		opts.MinClusterPixels = 1
	}

	// 1. Grid-based spatial clustering of changed pixels into disjoint bounding boxes
	clusters := clusterChangedPixels(a, b, opts)

	// 2. Correlate each cluster with DOM elements
	regions := make([]evidence.VisualRegion, 0, len(clusters))
	findings := make([]evidence.VisualFinding, 0, len(clusters))

	for i, cluster := range clusters {
		regionID := fmt.Sprintf("visual-region-%d", i+1)
		matchedElementIDs := findIntersectingElements(cluster.Bounds, elements, opts.MinOverlapRatio)

		vr := evidence.VisualRegion{
			ID:            regionID,
			Bounds:        cluster.Bounds,
			ChangedPixels: int64(cluster.PixelCount),
			DiffRatio:     float64(cluster.PixelCount) / float64(diffRes.Pixels),
			ElementIDs:    matchedElementIDs,
		}
		regions = append(regions, vr)

		severity := evidence.SeverityMedium
		if vr.DiffRatio > 0.05 || cluster.PixelCount > 500 {
			severity = evidence.SeverityHigh
		}

		finding := evidence.VisualFinding{
			ID:          fmt.Sprintf("finding:visualdiff:region-%d", i+1),
			Axis:        "visual_regression",
			Title:       fmt.Sprintf("Visual difference in region %d (%d changed pixels)", i+1, cluster.PixelCount),
			Description: fmt.Sprintf("Changed region bounds [%.0f, %.0f, %.0f, %.0f] intersects %d DOM elements", cluster.Bounds.X, cluster.Bounds.Y, cluster.Bounds.Width, cluster.Bounds.Height, len(matchedElementIDs)),
			Severity:    severity,
			Confidence:  1.0,
			Source:      "pixel_diff",
			RegionID:    regionID,
			ElementIDs:  matchedElementIDs,
		}
		findings = append(findings, finding)
	}

	return regions, findings, nil
}

type pixelCluster struct {
	Bounds     evidence.Rect
	PixelCount int
}

func clusterChangedPixels(a, b *image.RGBA, opts LocalizationOptions) []pixelCluster {
	width, height := a.Bounds().Dx(), a.Bounds().Dy()
	pad := opts.ClusterPadding
	gridSize := pad * 2
	if gridSize < 4 {
		gridSize = 4
	}

	cols := (width + gridSize - 1) / gridSize
	rows := (height + gridSize - 1) / gridSize
	grid := make([][]int, rows)
	for r := range grid {
		grid[r] = make([]int, cols)
	}

	// Step 1: Mark grid cells containing changed pixels
	for y := 0; y < height; y++ {
		ay := a.Rect.Min.Y + y
		by := b.Rect.Min.Y + y
		r := y / gridSize
		for x := 0; x < width; x++ {
			ax := a.Rect.Min.X + x
			bx := b.Rect.Min.X + x
			aoff := a.PixOffset(ax, ay)
			boff := b.PixOffset(bx, by)

			changed := false
			for c := 0; c < 4; c++ {
				if absDiff(a.Pix[aoff+c], b.Pix[boff+c]) > opts.ChannelTolerance {
					changed = true
					break
				}
			}
			if changed {
				c := x / gridSize
				grid[r][c]++
			}
		}
	}

	// Step 2: Group adjacent non-empty cells into connected components
	visited := make([][]bool, rows)
	for r := range visited {
		visited[r] = make([]bool, cols)
	}

	var clusters []pixelCluster

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if grid[r][c] == 0 || visited[r][c] {
				continue
			}

			// Flood fill connected component
			minX, minY := width, height
			maxX, maxY := 0, 0
			totalPixels := 0

			queue := [][2]int{{r, c}}
			visited[r][c] = true

			for len(queue) > 0 {
				curr := queue[0]
				queue = queue[1:]
				cr, cc := curr[0], curr[1]
				totalPixels += grid[cr][cc]

				cellMinX := cc * gridSize
				cellMinY := cr * gridSize
				cellMaxX := cellMinX + gridSize
				cellMaxY := cellMinY + gridSize

				if cellMaxX > width {
					cellMaxX = width
				}
				if cellMaxY > height {
					cellMaxY = height
				}

				if cellMinX < minX {
					minX = cellMinX
				}
				if cellMinY < minY {
					minY = cellMinY
				}
				if cellMaxX > maxX {
					maxX = cellMaxX
				}
				if cellMaxY > maxY {
					maxY = cellMaxY
				}

				// Check 4-connected neighbors
				neighbors := [][2]int{
					{cr - 1, cc},
					{cr + 1, cc},
					{cr, cc - 1},
					{cr, cc + 1},
				}
				for _, n := range neighbors {
					nr, nc := n[0], n[1]
					if nr >= 0 && nr < rows && nc >= 0 && nc < cols && !visited[nr][nc] && grid[nr][nc] > 0 {
						visited[nr][nc] = true
						queue = append(queue, [2]int{nr, nc})
					}
				}
			}

			if totalPixels >= opts.MinClusterPixels {
				clusters = append(clusters, pixelCluster{
					Bounds: evidence.Rect{
						X:      float64(minX),
						Y:      float64(minY),
						Width:  float64(maxX - minX),
						Height: float64(maxY - minY),
					},
					PixelCount: totalPixels,
				})
			}
		}
	}

	// If no clusters found above threshold but diff reported changes, fallback to single bounding box
	if len(clusters) == 0 {
		diffRes, _ := CompareRGBA(a, b, Options{ChannelTolerance: opts.ChannelTolerance})
		if diffRes.ChangedPixels > 0 {
			clusters = append(clusters, pixelCluster{
				Bounds: evidence.Rect{
					X:      float64(diffRes.Bounds.Min.X),
					Y:      float64(diffRes.Bounds.Min.Y),
					Width:  float64(diffRes.Bounds.Dx()),
					Height: float64(diffRes.Bounds.Dy()),
				},
				PixelCount: diffRes.ChangedPixels,
			})
		}
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].PixelCount > clusters[j].PixelCount
	})

	return clusters
}

func findIntersectingElements(region evidence.Rect, elements []evidence.ElementRef, minRatio float64) []string {
	matched := make([]string, 0)

	for _, elem := range elements {
		if !elem.Visible || elem.Bounds.Width <= 0 || elem.Bounds.Height <= 0 {
			continue
		}

		// Calculate 2D intersection
		ix := math.Max(0, math.Min(region.X+region.Width, elem.Bounds.X+elem.Bounds.Width)-math.Max(region.X, elem.Bounds.X))
		iy := math.Max(0, math.Min(region.Y+region.Height, elem.Bounds.Y+elem.Bounds.Height)-math.Max(region.Y, elem.Bounds.Y))
		interArea := ix * iy

		elemArea := elem.Bounds.Width * elem.Bounds.Height
		if elemArea > 0 && interArea/elemArea >= minRatio {
			matched = append(matched, elem.ID)
		}
	}

	sort.Strings(matched)
	return matched
}
