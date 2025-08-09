import { Settings2, MessageSquare, FileText, Hash, AlertCircle } from 'lucide-react';
import { usePreferencesStore, type ResponseStyle } from '../../stores/preferences.store';
import { Button } from '../ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from '../ui/dropdown-menu';
import { DropdownMenuTrigger } from '../ui/dropdown-menu';

export function ResponsePreferences() {
  const {
    responseStyle,
    maxResponseTokens,
    includeCodeComments,
    preferBulletPoints,
    setResponseStyle,
    setMaxResponseTokens,
    setIncludeCodeComments,
    setPreferBulletPoints,
  } = usePreferencesStore();

  const handleTokenLimitChange = (value: string) => {
    if (value === 'none') {
      setMaxResponseTokens(null);
    } else {
      setMaxResponseTokens(parseInt(value));
    }
  };

  const styleIcons: Record<ResponseStyle, JSX.Element> = {
    'ultra-concise': <Hash className="h-3 w-3" />,
    concise: <MessageSquare className="h-3 w-3" />,
    balanced: <FileText className="h-3 w-3" />,
    detailed: <FileText className="h-3 w-3" />,
  };

  const styleDescriptions: Record<ResponseStyle, string> = {
    'ultra-concise': '1-3 sentences max',
    concise: 'Brief, direct answers',
    balanced: 'Clear with context',
    detailed: 'Comprehensive explanations',
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="text-xs">
          <Settings2 className="h-3 w-3 mr-1" />
          {styleIcons[responseStyle]}
          <span className="ml-1 hidden sm:inline">{responseStyle}</span>
          {maxResponseTokens && (
            <span className="ml-1 text-amber-500" title={`Token limit: ${maxResponseTokens}`}>
              <AlertCircle className="h-3 w-3" />
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel>Response Preferences</DropdownMenuLabel>
        <DropdownMenuSeparator />
        
        <DropdownMenuLabel className="text-xs text-foreground-muted font-normal">
          Response Style
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={responseStyle}
          onValueChange={(value) => setResponseStyle(value as ResponseStyle)}
        >
          {(['ultra-concise', 'concise', 'balanced', 'detailed'] as ResponseStyle[]).map((style) => (
            <DropdownMenuRadioItem key={style} value={style}>
              <div className="flex items-center justify-between w-full">
                <div className="flex items-center gap-2">
                  {styleIcons[style]}
                  <span className="capitalize">{style}</span>
                </div>
                <span className="text-xs text-foreground-muted ml-2">
                  {styleDescriptions[style]}
                </span>
              </div>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        
        <DropdownMenuSeparator />
        
        <DropdownMenuLabel className="text-xs text-foreground-muted font-normal">
          Token Limit
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={maxResponseTokens?.toString() || 'none'}
          onValueChange={handleTokenLimitChange}
        >
          <DropdownMenuRadioItem value="none">No limit</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="150">150 tokens (~100 words)</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="300">300 tokens (~225 words)</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="500">500 tokens (~375 words)</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="1000">1000 tokens (~750 words)</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="2000">2000 tokens (~1500 words)</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
        
        <DropdownMenuSeparator />
        
        <DropdownMenuLabel className="text-xs text-foreground-muted font-normal">
          Formatting Options
        </DropdownMenuLabel>
        <DropdownMenuCheckboxItem
          checked={preferBulletPoints}
          onCheckedChange={setPreferBulletPoints}
        >
          Prefer bullet points
        </DropdownMenuCheckboxItem>
        <DropdownMenuCheckboxItem
          checked={includeCodeComments}
          onCheckedChange={setIncludeCodeComments}
        >
          Include code comments
        </DropdownMenuCheckboxItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}