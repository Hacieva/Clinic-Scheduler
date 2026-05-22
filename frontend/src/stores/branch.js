import { create } from 'zustand'

const useBranchStore = create((set) => ({
  activeBranchId: null,  // null = all branches
  setActiveBranchId: (id) => set({ activeBranchId: id }),
}))

export default useBranchStore
