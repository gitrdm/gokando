#!/usr/bin/env python3
"""
Validate the current solution from the graph coloring example
"""

# Current solution from the example output
solution = {
    'WA': 'blue',
    'NT': 'green', 
    'SA': 'red',
    'Q': 'blue',
    'NSW': 'green',
    'V': 'blue',
    'T': 'red'
}

# Define adjacencies (who borders whom)
adjacencies = {
    'WA': ['NT', 'SA'],
    'NT': ['WA', 'SA', 'Q'],
    'SA': ['WA', 'NT', 'Q', 'NSW', 'V'],
    'Q': ['NT', 'SA', 'NSW'],
    'NSW': ['Q', 'SA', 'V'],
    'V': ['SA', 'NSW'],
    'T': []  # Tasmania is an island
}

def validate_coloring(solution, adjacencies):
    """Check if the coloring satisfies all adjacency constraints."""
    valid = True
    violations = []
    
    for region, neighbors in adjacencies.items():
        region_color = solution[region]
        print(f"{region}({region_color}): neighbors = {', '.join(f'{n}({solution[n]})' for n in neighbors)}", end="")
        
        # Check if any neighbor has the same color
        for neighbor in neighbors:
            if solution[neighbor] == region_color:
                valid = False
                violations.append(f"{region} and {neighbor} both have color {region_color}")
                print(" ❌")
                break
        else:
            print(" ✓")
    
    return valid, violations

print("Validating current solution from graph coloring example:")
print("=" * 50)

valid, violations = validate_coloring(solution, adjacencies)

print("\nResult:", "✅ Valid 3-coloring of Australia" if valid else f"❌ Invalid coloring: {violations}")

# Count colors used
colors_used = set(solution.values())
print(f"Colors used: {len(colors_used)} - {sorted(colors_used)}")