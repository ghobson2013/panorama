package render

import (
	"github.com/weqqr/panorama/pkg/spatial"
	"github.com/weqqr/panorama/pkg/world"
)

type BlockNeighborhood struct {
	blocks [27]*world.MapBlock
}

var neighborhoodCenter = spatial.BlockPos{X: 1, Y: 1, Z: 1}

func (b *BlockNeighborhood) FetchBlock(w *world.World, posOffset, worldPos spatial.BlockPos) {
	block, err := w.GetBlock(worldPos.Add(posOffset))

	if err != nil {
		return
	}

	b.SetBlock(neighborhoodCenter.Add(posOffset), block)
}

func (b *BlockNeighborhood) SetBlock(pos spatial.BlockPos, block *world.MapBlock) {
	b.blocks[pos.X*9+pos.Y*3+pos.Z] = block
}

func (b *BlockNeighborhood) getBlockByNodePos(pos spatial.NodePos) *world.MapBlock {
	bx := pos.X/spatial.BlockSize + neighborhoodCenter.X
	by := pos.Y/spatial.BlockSize + neighborhoodCenter.Y
	bz := pos.Z/spatial.BlockSize + neighborhoodCenter.Z

	return b.blocks[bz*9+by*3+bx]
}

func (b *BlockNeighborhood) GetNode(pos spatial.NodePos) (string, uint8, uint8) {
	block := b.getBlockByNodePos(pos)

	if block == nil {
		return "air", 0, 0
	}

	node := block.GetNode(spatial.NodePos{
		X: pos.X % spatial.BlockSize,
		Y: pos.Y % spatial.BlockSize,
		Z: pos.Z % spatial.BlockSize,
	})
	name := block.ResolveName(node.ID)
	return name, node.Param1, node.Param2
}

func (b *BlockNeighborhood) GetParam1(pos spatial.NodePos) uint8 {
	block := b.getBlockByNodePos(pos)

	if block == nil {
		return 0
	}

	node := block.GetNode(spatial.NodePos{
		X: pos.X % spatial.BlockSize,
		Y: pos.Y % spatial.BlockSize,
		Z: pos.Z % spatial.BlockSize,
	})

	return node.Param1
}