#!/usr/bin/env python3
from PIL import Image, ImageDraw, ImageFont
import os

# Create icon sizes
sizes = [
    (32, "32x32.png"),
    (128, "128x128.png"),
    (256, "128x128@2x.png")
]

icon_dir = "src-tauri/icons"

for size, filename in sizes:
    # Create a new image with a gradient background
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    
    # Draw gradient background
    for y in range(size):
        color_value = int(10 + (y / size) * 20)  # Gradient from #0a0a0a to #1e1e1e
        draw.rectangle([(0, y), (size, y+1)], fill=(color_value, color_value, color_value, 255))
    
    # Draw a rounded rectangle border
    padding = size // 8
    draw.rounded_rectangle(
        [(padding, padding), (size-padding, size-padding)],
        radius=size//8,
        outline=(32, 145, 246, 255),  # Accent blue
        width=max(1, size//32)
    )
    
    # Draw "AX" text
    try:
        font_size = size // 3
        # Try to use a system font, fall back to default if not available
        try:
            font = ImageFont.truetype("/System/Library/Fonts/Helvetica.ttc", font_size)
        except:
            font = ImageFont.load_default()
        
        text = "AX"
        bbox = draw.textbbox((0, 0), text, font=font)
        text_width = bbox[2] - bbox[0]
        text_height = bbox[3] - bbox[1]
        
        x = (size - text_width) // 2
        y = (size - text_height) // 2 - size // 16
        
        draw.text((x, y), text, fill=(255, 255, 255, 255), font=font)
    except:
        # If text drawing fails, just draw a simple X
        margin = size // 4
        draw.line([(margin, margin), (size-margin, size-margin)], fill=(255, 255, 255, 255), width=max(1, size//16))
        draw.line([(size-margin, margin), (margin, size-margin)], fill=(255, 255, 255, 255), width=max(1, size//16))
    
    # Save the image
    img.save(os.path.join(icon_dir, filename))
    print(f"Created {filename}")

# Create ICO file for Windows (using the 32x32 icon)
try:
    img_32 = Image.open(os.path.join(icon_dir, "32x32.png"))
    img_32.save(os.path.join(icon_dir, "icon.ico"), format='ICO', sizes=[(32, 32)])
    print("Created icon.ico")
except Exception as e:
    print(f"Could not create ICO: {e}")

# Create a placeholder ICNS file for macOS (we'll use the PNG for now)
import shutil
shutil.copy(os.path.join(icon_dir, "128x128.png"), os.path.join(icon_dir, "icon.icns"))
print("Created icon.icns (placeholder)")

print("\nIcon generation complete!")